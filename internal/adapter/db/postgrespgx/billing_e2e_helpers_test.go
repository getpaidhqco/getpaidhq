//go:build integration

// Shared, non-Test helper suite for the billing / usage e2e tests, targeting the
// pgx adapter. Every seeder, fixture, service builder, and noop stub the ported
// e2e TEST functions rely on lives here.
//
// Storage notes:
//   - repos take a *pgxpool.Pool; construct with the NewXxxRepo names.
//   - rows are seeded through the repository PORT Create methods on domain objects;
//     the pgx Create methods already map empty-string FKs to NULL, so no
//     nullable-column omit tricks are needed.
//   - targeted single-column updates use raw pool.Exec statements.
//   - poolForTest(t) (defined in customer_repo_test.go) replaces testDB(t); it boots
//     the shared testcontainer and opens a pool. seedOrgForTest / pgxRepoSet already
//     exist in this package and are NOT redefined here.
package postgrespgx

import (
	"context"
	"fmt"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/adapter/memory"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib/errors"
	"getpaidhq/internal/lib/ids"
)

// ---------------------------------------------------------------------------
// noop stubs (storage-agnostic)
// ---------------------------------------------------------------------------

// noopLogger is a Logger that drops everything. The charge path is chatty
// (Infof/Errorf on every step) but the tests assert on returned state, not
// logs, so silence keeps the output readable. Panicf must actually halt to
// preserve the interface contract relied on elsewhere.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any)      {}
func (noopLogger) Info(string, ...any)       {}
func (noopLogger) Warn(string, ...any)       {}
func (noopLogger) Error(string, ...any)      {}
func (noopLogger) Fatal(string, ...any)      {}
func (noopLogger) Debugf(string, ...any)     {}
func (noopLogger) Infof(string, ...any)      {}
func (noopLogger) Warnf(string, ...any)      {}
func (noopLogger) Errorf(string, ...any)     {}
func (noopLogger) Panicf(t string, a ...any) { panic(t) }
func (noopLogger) Fatalf(string, ...any)     {}
func (noopLogger) Sync() error               { return nil }

// noopPubSub satisfies port.PubSub without any transport. SubscriptionService's
// constructor subscribes to "subscription.workflow.>", and the charge handlers
// publish success/failure events; none of that is asserted here, so every
// method is a no-op. Subscribe returns a real (no-op) subscription so the
// constructor's nil-check on the returned subscription is satisfied.
type noopPubSub struct{}

func (noopPubSub) Publish(context.Context, string, string, any) error { return nil }
func (noopPubSub) Subscribe(string, func(string, []byte)) (port.PubSubSubscription, error) {
	return noopSubscription{}, nil
}
func (noopPubSub) Close() error { return nil }

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() error { return nil }

// noopEngine satisfies port.Engine without orchestration. CompleteOrder's
// post-commit StartSubscriptionWorkflow call logs (not returns) on error, and
// the coupon/order tests drive the billing cycles by hand, so the engine is inert.
type noopEngine struct{}

func (noopEngine) StartWorkflow(context.Context, port.WorkflowType, any) (port.WorkflowResult, error) {
	return port.WorkflowResult{}, nil
}
func (noopEngine) StartSubscriptionWorkflow(context.Context, domain.Subscription) error { return nil }
func (noopEngine) UpdateSubscriptionWorkflow(context.Context, string, domain.Subscription) error {
	return nil
}
func (noopEngine) CancelSubscriptionWorkflow(context.Context, domain.Subscription) error { return nil }
func (noopEngine) SignalSubscriptionWorkflow(context.Context, string, domain.Subscription, any) error {
	return nil
}

// noopPriorPayments backs the FirstTimeTransaction restriction; the coupons in
// these tests have no such restriction, so it is never consulted.
type noopPriorPayments struct{}

func (noopPriorPayments) HasPriorSuccessfulPayment(context.Context, string, string) (bool, error) {
	return false, nil
}

// noopOrderInvoicing satisfies service.OrderInvoicing without building an
// invoice: BuildForOrder returns port.ErrNotFound (order with nothing to
// invoice), so order completion does no invoicing — the behaviour the
// coupon-billing flow asserts (it does not opt into order-level invoicing; the
// per-cycle invoices are produced later by the billing tail).
type noopOrderInvoicing struct{}

func (noopOrderInvoicing) BuildForOrder(ctx context.Context, order domain.Order) (domain.Invoice, error) {
	return domain.Invoice{}, port.ErrNotFound
}

func (noopOrderInvoicing) MarkOpen(ctx context.Context, orgId, invoiceId string) (domain.Invoice, error) {
	return domain.Invoice{}, nil
}

func (noopOrderInvoicing) SettleOrderInvoice(ctx context.Context, orgId, invoiceId string) error {
	return nil
}

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------

// subFixture seeds the parent chain (customer, price, order, order item) the
// subscription's foreign keys point at, and returns a ready-to-create
// Subscription wired to them.
type subFixture struct {
	customer domain.Customer
	order    domain.Order
	item     domain.OrderItem
	sub      domain.Subscription
}

// meteredFixture is the parent chain for a usage-based subscription: a customer
// (with an ExternalId so external_customer_id matching can be tested), a meter,
// a metered price, and an order/order-item/subscription wired to them. The order
// carries a single metered item, so the subscription is itself the "primary"
// that bills the order's usage.
type meteredFixture struct {
	customer domain.Customer
	meter    domain.BillableMetric
	price    domain.Price
	order    domain.Order
	item     domain.OrderItem
	sub      domain.Subscription
}

// graduatedEmailFixture is the parent chain for a graduated, usage-based email
// subscription: a customer, a SUM meter over the "emails" field, a graduated
// metered price carrying emailTiers(), and an order/order-item/subscription wired
// to them. The subscription owns the metered line, so it bills the order's usage.
type graduatedEmailFixture struct {
	customer domain.Customer
	meter    domain.BillableMetric
	price    domain.Price
	order    domain.Order
	item     domain.OrderItem
	sub      domain.Subscription
}

// meteredUnitPriceCents is the metered rate: every counted event costs this many cents.
const meteredUnitPriceCents = 10

// ---------------------------------------------------------------------------
// org seeding & cleanup
// ---------------------------------------------------------------------------

// seedOrg inserts a minimal org row so FK constraints on child tables are satisfied.
func seedOrg(t *testing.T, pool *pgxpool.Pool, orgId string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	_, err := NewOrgRepo(pool).Create(context.Background(), domain.Org{
		Id:        orgId,
		Name:      "Test Org " + orgId,
		Country:   "US",
		Timezone:  "UTC",
		Status:    domain.OrgStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)
}

// uniqueOrg generates a unique org ID and seeds the matching org row so that
// FK constraints on all child tables (customers, orders, api_keys, …) are satisfied.
func uniqueOrg(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	orgId := ids.Generate("org_test")
	seedOrg(t, pool, orgId)
	return orgId
}

// cleanupOrg registers a t.Cleanup that raw-deletes the org's rows in FK-safe
// order. Deletes are best-effort (errors ignored).
func cleanupOrg(t *testing.T, pool *pgxpool.Pool, orgId string) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		// FK-safe delete order (children before parents). Note the psp config
		// lives in the `gateways` table.
		ordered := []string{
			"dunning_communications",
			"payment_update_tokens",
			"dunning_attempts",
			"dunning_campaigns",
			"dunning_configurations",
			"customer_dunning_history",
			"refunds",
			"payments",
			"subscriptions",
			"order_items",
			"orders",
			"carts",
			"prices",
			"products",
			"payment_methods",
			"customer_cohorts",
			"cohorts",
			"customers",
			"settings",
			"gateways",
			"api_keys",
		}
		for _, tbl := range ordered {
			_, _ = pool.Exec(ctx, "DELETE FROM "+tbl+" WHERE org_id=$1", orgId)
		}
		// The orgs table is keyed by `id`, not `org_id`.
		_, _ = pool.Exec(ctx, "DELETE FROM orgs WHERE id=$1", orgId)
	})
}

// ---------------------------------------------------------------------------
// entity seeders
// ---------------------------------------------------------------------------

func seedCustomer(t *testing.T, pool *pgxpool.Pool, orgId string) domain.Customer {
	t.Helper()
	c := domain.Customer{
		OrgId:     orgId,
		Id:        ids.Generate("cus"),
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     fmt.Sprintf("%s@example.com", ids.Generate("ada")),
		Phone:     "+155****1111",
		BillingAddress: domain.Address{
			Line1:   "1 Analytical Engine Way",
			City:    "London",
			Country: "GB",
		},
		Metadata:  map[string]string{"tier": "gold"},
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	// default_payment_method_id has a FK constraint; the pgx Create maps the empty
	// string to NULL, so no column-omit trick is needed.
	_, err := NewCustomerRepo(pool).Create(context.Background(), c)
	require.NoError(t, err)
	return c
}

// seedVariantChain seeds a product + variant (required parent chain for prices)
// and returns the variant id. Callers must set price.VariantId to the returned value.
func seedVariantChain(t *testing.T, pool *pgxpool.Pool, orgId string) string {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	productId := ids.Generate("prod")
	_, err := NewProductRepo(pool).Create(context.Background(), domain.Product{
		OrgId:     orgId,
		Id:        productId,
		Name:      "Test Product",
		Status:    domain.ProductStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)
	variantId := ids.Generate("var")
	_, err = NewVariantRepo(pool).Create(context.Background(), domain.Variant{
		OrgId:     orgId,
		Id:        variantId,
		ProductId: productId,
		Name:      "Default",
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)
	return variantId
}

func seedPrice(t *testing.T, pool *pgxpool.Pool, orgId string) domain.Price {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	// prices.variant_id is NOT NULL with a FK to variants → products → orgs.
	variantId := seedVariantChain(t, pool, orgId)
	p := domain.Price{
		OrgId:              orgId,
		Id:                 ids.Generate("price"),
		VariantId:          variantId,
		Label:              "Monthly Pro",
		Category:           domain.PriceCategorySubscription,
		Scheme:             domain.Fixed,
		Currency:           domain.USD,
		UnitPrice:          1999,
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, err := NewPriceRepo(pool).Create(context.Background(), p)
	require.NoError(t, err)
	return p
}

func seedOrderItem(t *testing.T, pool *pgxpool.Pool, orgId, orderId, priceId string) domain.OrderItem {
	t.Helper()
	item := domain.OrderItem{
		OrgId:       orgId,
		Id:          ids.Generate("oi"),
		OrderId:     orderId,
		PriceId:     priceId,
		Description: "Monthly Pro",
		Quantity:    1,
		Subtotal:    1999,
		Total:       1999,
		// metadata is NOT NULL in the DB; product_id is NOT NULL but has no FK.
		Metadata:  map[string]string{},
		ProductId: "test-product",
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	// variant_id is nullable with a FK to variants; the pgx Create maps the empty
	// string to NULL so the FK constraint is satisfied without a real variant parent.
	_, err := NewOrderRepo(pool).CreateOrderItem(context.Background(), item)
	require.NoError(t, err)
	return item
}

func seedOrder(t *testing.T, pool *pgxpool.Pool, orgId, customerId string) domain.Order {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	// orders.cart_id is NOT NULL with a FK to carts(org_id, id). Seed a minimal cart.
	cartId := ids.Generate("cart")
	_, err := NewCartRepo(pool).Create(context.Background(), domain.Cart{
		OrgId:     orgId,
		Id:        cartId,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)
	o := domain.Order{
		OrgId:      orgId,
		Id:         ids.Generate("ord"),
		CustomerId: customerId,
		CartId:     cartId,
		Reference:  "REF-" + ids.Generate("r"),
		Status:     domain.OrderStatusPending,
		Currency:   "USD",
		Total:      1999,
		Metadata:   map[string]string{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err = NewOrderRepo(pool).Create(context.Background(), o)
	require.NoError(t, err)
	return o
}

// seedCoupon inserts a valid percentage coupon for orgId and returns it.
func seedCoupon(t *testing.T, pool *pgxpool.Pool, orgId string) domain.Coupon {
	t.Helper()
	in, err := domain.NewCoupon(domain.NewCouponInput{
		OrgId:        orgId,
		Name:         "Seed Coupon",
		DiscountType: domain.DiscountTypePercentage,
		PercentOff:   decimal.NewFromInt(25),
		Duration:     domain.DurationForever,
	})
	require.NoError(t, err)
	c, err := NewCouponRepo(pool).Create(context.Background(), in)
	require.NoError(t, err)
	return c
}

// seedMemoryPsp configures the org so the GatewayFactory resolves to the memory
// gateway. The factory reads gateways.FindById(orgId, id) and dispatches on the
// row's PspId; the memory gateway needs no credentials, so the credentials
// column stays empty. The subscription's PspId must equal `pspConfigId` so
// ChargeForBillingPeriod's NewGateway(orgId, string(sub.PspId)) lookup hits this row.
func seedMemoryPsp(t *testing.T, pool *pgxpool.Pool, orgId string) string {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	pspConfigId := ids.Generate("gw")
	_, err := NewPspRepo(pool).Create(context.Background(), domain.PspConfig{
		OrgId:     orgId,
		Id:        pspConfigId,
		PspId:     domain.Memory,
		Name:      "Memory (test)",
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)
	return pspConfigId
}

// seedPaymentMethod creates an active card the recurring charge can reference.
// ChargeForBillingPeriod fetches the payment method by sub.PaymentMethodId, so
// the sub must point at this row.
func seedPaymentMethod(t *testing.T, pool *pgxpool.Pool, orgId, customerId string) domain.PaymentMethod {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	pm := domain.PaymentMethod{
		OrgId:      orgId,
		Id:         ids.Generate("pm"),
		Status:     domain.PaymentMethodStatusActive,
		Psp:        string(domain.Memory),
		Name:       "Visa ****4242",
		CustomerId: customerId,
		Type:       domain.PaymentMethodTypeCard,
		Token:      domain.Secret(ids.Generate("tok")),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err := NewPaymentMethodRepo(pool).Create(context.Background(), pm)
	require.NoError(t, err)
	return pm
}

// seedMemoryPspForSub wires the subscription to a memory gateway + active payment
// method, persisting the psp_id / payment_method_id columns so the re-read inside
// ChargeForBillingPeriod resolves them, and updating the in-memory sub the caller holds.
func seedMemoryPspForSub(t *testing.T, pool *pgxpool.Pool, orgId string, sub *domain.Subscription) {
	t.Helper()
	pspConfigId := seedMemoryPsp(t, pool, orgId)
	pm := seedPaymentMethod(t, pool, orgId, sub.CustomerId)
	sub.PspId = domain.Gateway(pspConfigId)
	sub.PaymentMethodId = pm.Id
	_, err := pool.Exec(context.Background(),
		"UPDATE subscriptions SET psp_id=$1, payment_method_id=$2 WHERE org_id=$3 AND id=$4",
		pspConfigId, pm.Id, orgId, sub.Id)
	require.NoError(t, err)
}

// seedDecliningCard wires the subscription to the memory gateway with a payment
// method carrying the gateway's decline token, so every charge fails as a
// retryable card error.
func seedDecliningCard(t *testing.T, pool *pgxpool.Pool, orgId string, sub *domain.Subscription) {
	t.Helper()
	pspConfigId := seedMemoryPsp(t, pool, orgId)
	now := time.Now().UTC().Truncate(time.Microsecond)
	pm := domain.PaymentMethod{
		OrgId:      orgId,
		Id:         ids.Generate("pm"),
		Status:     domain.PaymentMethodStatusActive,
		Psp:        string(domain.Memory),
		Name:       "Visa ****0002 (declines)",
		CustomerId: sub.CustomerId,
		Type:       domain.PaymentMethodTypeCard,
		Token:      memory.DeclineToken,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err := NewPaymentMethodRepo(pool).Create(context.Background(), pm)
	require.NoError(t, err)
	sub.PspId = domain.Gateway(pspConfigId)
	sub.PaymentMethodId = pm.Id
	_, err = pool.Exec(context.Background(),
		"UPDATE subscriptions SET psp_id=$1, payment_method_id=$2 WHERE org_id=$3 AND id=$4",
		pspConfigId, pm.Id, orgId, sub.Id)
	require.NoError(t, err)
}

// seedOneTimePrice seeds a one-time (non-recurring) price of unitPrice cents on a
// fresh product/variant, returning the product and price ids. A one-time line
// starts no subscription — it is billed once on the combined invoice.
func seedOneTimePrice(t *testing.T, pool *pgxpool.Pool, orgId string, unitPrice int64) (productId, priceId string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	variantId := seedVariantChain(t, pool, orgId)
	v, err := NewVariantRepo(pool).FindById(context.Background(), orgId, variantId)
	require.NoError(t, err)
	productId = v.ProductId

	p := domain.Price{
		OrgId:              orgId,
		Id:                 ids.Generate("price"),
		VariantId:          variantId,
		Label:              "Setup fee",
		Category:           domain.OneTime,
		Scheme:             domain.Fixed,
		Currency:           domain.USD,
		UnitPrice:          unitPrice,
		UnitCount:          1,
		BillingInterval:    domain.BillingIntervalNone,
		BillingIntervalQty: 0,
		TrialInterval:      domain.BillingIntervalNone,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, err = NewPriceRepo(pool).Create(context.Background(), p)
	require.NoError(t, err)
	return productId, p.Id
}

// seedSubscriptionPrice seeds a product + variant + a fixed $100/cycle
// subscription price capped at `cycles` cycles, billed every minute, and
// returns the product and price.
func seedSubscriptionPrice(t *testing.T, pool *pgxpool.Pool, orgId string, cycles int) (productId, priceId string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	variantId := seedVariantChain(t, pool, orgId)
	// seedVariantChain creates its own product; reuse that product as the cart's.
	v, err := NewVariantRepo(pool).FindById(context.Background(), orgId, variantId)
	require.NoError(t, err)
	productId = v.ProductId

	p := domain.Price{
		OrgId:              orgId,
		Id:                 ids.Generate("price"),
		VariantId:          variantId,
		Label:              "Warp Drive Pro",
		Category:           domain.PriceCategorySubscription,
		Scheme:             domain.Fixed,
		Currency:           domain.USD,
		UnitPrice:          10000, // $100.00
		UnitCount:          1,
		BillingInterval:    domain.BillingIntervalMinute,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		Cycles:             cycles,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, err = NewPriceRepo(pool).Create(context.Background(), p)
	require.NoError(t, err)
	return productId, p.Id
}

// ---------------------------------------------------------------------------
// subscription fixtures
// ---------------------------------------------------------------------------

func newSubscription(orgId, customerId, orderId string) domain.Subscription {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return domain.Subscription{
		OrgId:              orgId,
		Id:                 ids.Generate("sub"),
		PspId:              domain.Paystack,
		OrderId:            orderId,
		CustomerId:         customerId,
		Status:             domain.SubscriptionStatusActive,
		StartDate:          now,
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		Cycles:             12,
		Currency:           "USD",
		Metadata:           map[string]string{"plan": "pro"},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func seedSubFixture(t *testing.T, pool *pgxpool.Pool, orgId string) subFixture {
	t.Helper()
	cust := seedCustomer(t, pool, orgId)
	price := seedPrice(t, pool, orgId)
	order := seedOrder(t, pool, orgId, cust.Id)
	item := seedOrderItem(t, pool, orgId, order.Id, price.Id)
	return subFixture{customer: cust, order: order, item: item, sub: newSubscription(orgId, cust.Id, order.Id)}
}

// seedMeteredFixture persists a metered subscription due for the given period.
// The subscription is active, cycle 0, with explicit period boundaries so usage
// window scoping can be asserted exactly.
func seedMeteredFixture(t *testing.T, pool *pgxpool.Pool, orgId string, periodStart, periodEnd time.Time) meteredFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)

	cust := domain.Customer{
		OrgId:      orgId,
		Id:         ids.Generate("cus"),
		ExternalId: ids.Generate("ext_cus"), // the merchant's own id, matched on usage events
		FirstName:  "Grace",
		LastName:   "Hopper",
		Email:      ids.Generate("grace") + "@example.com",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err := NewCustomerRepo(pool).Create(context.Background(), cust)
	require.NoError(t, err)

	meter := domain.BillableMetric{
		OrgId:       orgId,
		Id:          ids.Generate("met"),
		Code:        "api_calls",
		Name:        "API Calls",
		Aggregation: domain.AggregationCount,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err = NewMeterRepo(pool).Create(context.Background(), meter)
	require.NoError(t, err)

	variantId := seedVariantChain(t, pool, orgId)
	price := domain.Price{
		OrgId:              orgId,
		Id:                 ids.Generate("price"),
		VariantId:          variantId,
		Label:              "Metered API",
		Category:           domain.PriceCategorySubscription,
		Scheme:             domain.Fixed,
		Currency:           domain.USD,
		UnitPrice:          meteredUnitPriceCents,
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		BillableMetricId:   meter.Id, // <- makes the price metered
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, err = NewPriceRepo(pool).Create(context.Background(), price)
	require.NoError(t, err)

	order := seedOrder(t, pool, orgId, cust.Id)
	item := seedOrderItem(t, pool, orgId, order.Id, price.Id)

	sub := domain.Subscription{
		OrgId:              orgId,
		Id:                 ids.Generate("sub"),
		PspId:              domain.Paystack,
		OrderId:            order.Id,
		CustomerId:         cust.Id,
		Status:             domain.SubscriptionStatusActive,
		Currency:           "USD",
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		Cycles:             12,
		CyclesProcessed:    0,
		StartDate:          periodStart,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		RenewsAt:           periodEnd,
		Metadata:           map[string]string{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	// payment_method_id is nullable with FK; the pgx Create maps empty → NULL.
	_, err = NewSubscriptionRepo(pool).Create(context.Background(), sub)
	require.NoError(t, err)
	// The subscription owns its metered line.
	_, err = pool.Exec(context.Background(),
		"UPDATE order_items SET subscription_id=$1 WHERE org_id=$2 AND id=$3",
		sub.Id, orgId, item.Id)
	require.NoError(t, err)

	return meteredFixture{customer: cust, meter: meter, price: price, order: order, item: item, sub: sub}
}

// seedUsageFixture persists the parent chain for a usage-based subscription —
// customer, the GIVEN meter and price (wired together), order, order item, and an
// active subscription due for [periodStart, periodEnd) — and returns the chain.
// It is seedMeteredFixture generalised to any meter/price configuration.
func seedUsageFixture(t *testing.T, pool *pgxpool.Pool, orgId string, meter domain.BillableMetric, price domain.Price, periodStart, periodEnd time.Time) meteredFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)

	cust := domain.Customer{
		OrgId:      orgId,
		Id:         ids.Generate("cus"),
		ExternalId: ids.Generate("ext_cus"),
		FirstName:  "Ada",
		LastName:   "Lovelace",
		Email:      ids.Generate("ada") + "@example.com",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err := NewCustomerRepo(pool).Create(context.Background(), cust)
	require.NoError(t, err)

	meter.OrgId = orgId
	meter.Id = ids.Generate("met")
	meter.CreatedAt, meter.UpdatedAt = now, now
	_, err = NewMeterRepo(pool).Create(context.Background(), meter)
	require.NoError(t, err)

	price.OrgId = orgId
	price.Id = ids.Generate("price")
	price.VariantId = seedVariantChain(t, pool, orgId)
	price.Category = domain.PriceCategorySubscription
	price.Currency = domain.USD
	price.BillingInterval = domain.BillingIntervalMonth
	price.BillingIntervalQty = 1
	price.TrialInterval = domain.BillingIntervalNone
	price.BillableMetricId = meter.Id
	price.CreatedAt, price.UpdatedAt = now, now
	_, err = NewPriceRepo(pool).Create(context.Background(), price)
	require.NoError(t, err)

	order := seedOrder(t, pool, orgId, cust.Id)
	item := seedOrderItem(t, pool, orgId, order.Id, price.Id)

	sub := domain.Subscription{
		OrgId:              orgId,
		Id:                 ids.Generate("sub"),
		PspId:              domain.Paystack,
		OrderId:            order.Id,
		CustomerId:         cust.Id,
		Status:             domain.SubscriptionStatusActive,
		Currency:           "USD",
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		Cycles:             12,
		CyclesProcessed:    0,
		StartDate:          periodStart,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		RenewsAt:           periodEnd,
		Metadata:           map[string]string{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, err = NewSubscriptionRepo(pool).Create(context.Background(), sub)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(),
		"UPDATE order_items SET subscription_id=$1 WHERE org_id=$2 AND id=$3",
		sub.Id, orgId, item.Id)
	require.NoError(t, err)

	return meteredFixture{customer: cust, meter: meter, price: price, order: order, item: item, sub: sub}
}

// emailTiers is the graduated email-API ladder (cents per email; sub-cent rates
// are exact via decimal). Each slice is billed at its own rate (marginal/graduated):
//
//	tier 1:        0 – 10,000 emails   $0.0010 each  (0.1¢)
//	tier 2:   10,001 – 100,000 emails  $0.0005 each  (0.05¢)
//	tier 3:  100,001 +          emails  $0.0002 each  (0.02¢)
func emailTiers() []domain.PriceTier {
	d := decimal.RequireFromString
	return []domain.PriceTier{
		{FromValue: d("0"), ToValue: d("10000"), PerUnitAmount: d("0.1")},
		{FromValue: d("10000"), ToValue: d("100000"), PerUnitAmount: d("0.05")},
		{FromValue: d("100000"), ToValue: d("0"), PerUnitAmount: d("0.02")}, // 0 = unbounded
	}
}

func seedGraduatedEmailFixture(t *testing.T, pool *pgxpool.Pool, orgId string, periodStart, periodEnd time.Time) graduatedEmailFixture {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)

	cust := domain.Customer{
		OrgId:      orgId,
		Id:         ids.Generate("cus"),
		ExternalId: ids.Generate("ext_cus"),
		FirstName:  "Ada",
		LastName:   "Lovelace",
		Email:      ids.Generate("ada") + "@example.com",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err := NewCustomerRepo(pool).Create(context.Background(), cust)
	require.NoError(t, err)

	// SUM meter: each event reports a batch of emails via metadata["emails"]; the
	// period quantity is the sum of those batches.
	meter := domain.BillableMetric{
		OrgId:       orgId,
		Id:          ids.Generate("met"),
		Code:        "emails_sent",
		Name:        "Emails Sent",
		Aggregation: domain.AggregationSum,
		FieldName:   "emails",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err = NewMeterRepo(pool).Create(context.Background(), meter)
	require.NoError(t, err)

	variantId := seedVariantChain(t, pool, orgId)
	// Graduated price. UnitPrice is intentionally left zero — PriceUsage switches on
	// Scheme=Graduated and prices purely from Tiers; a non-zero UnitPrice here would
	// be ignored, and asserting $85.00 proves the tier path (not the flat path) ran.
	price := domain.Price{
		OrgId:              orgId,
		Id:                 ids.Generate("price"),
		VariantId:          variantId,
		Label:              "Transactional Email",
		Category:           domain.PriceCategorySubscription,
		Scheme:             domain.Graduated,
		Currency:           domain.USD,
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		BillableMetricId:   meter.Id, // <- makes the price metered
		Tiers:              emailTiers(),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, err = NewPriceRepo(pool).Create(context.Background(), price)
	require.NoError(t, err)

	order := seedOrder(t, pool, orgId, cust.Id)
	item := seedOrderItem(t, pool, orgId, order.Id, price.Id)

	sub := domain.Subscription{
		OrgId:              orgId,
		Id:                 ids.Generate("sub"),
		PspId:              domain.Paystack,
		OrderId:            order.Id,
		CustomerId:         cust.Id,
		Status:             domain.SubscriptionStatusActive,
		Currency:           "USD",
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingIntervalNone,
		Cycles:             12,
		CyclesProcessed:    0,
		StartDate:          periodStart,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		RenewsAt:           periodEnd,
		Metadata:           map[string]string{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_, err = NewSubscriptionRepo(pool).Create(context.Background(), sub)
	require.NoError(t, err)
	// The subscription owns its metered line.
	_, err = pool.Exec(context.Background(),
		"UPDATE order_items SET subscription_id=$1 WHERE org_id=$2 AND id=$3",
		sub.Id, orgId, item.Id)
	require.NoError(t, err)

	return graduatedEmailFixture{customer: cust, meter: meter, price: price, order: order, item: item, sub: sub}
}

// seedPastDueSubscriptionWithOpenInvoice drives a fixture through one failed
// charge so that the subscription is past_due with Retries = 1 and an open invoice
// for cycle 0. Returns the updated (past_due) subscription so callers can cancel it.
func seedPastDueSubscriptionWithOpenInvoice(t *testing.T, orgId string) domain.Subscription {
	t.Helper()
	pool := poolForTest(t)
	ctx := context.Background()

	jan1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fx := seedUsageFixture(t, pool, orgId,
		domain.BillableMetric{Code: "api_calls_cis", Name: "API calls", Aggregation: domain.AggregationCount},
		domain.Price{Label: "API", Scheme: domain.Fixed, UnitPrice: 10},
		jan1, feb1)
	seedDecliningCard(t, pool, orgId, &fx.sub)

	usage := buildUsageService(t, pool)
	recordUsage(t, usage, port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: "cis_ev1", Timestamp: jan1.Add(time.Hour),
	})

	svc := buildSubscriptionService(t, pool)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusFailed, result.Status)

	updated, err := svc.HandleSubscriptionChargeFailure(ctx, port.SubscriptionChargeInput{
		Subscription: fx.sub, ChargeResult: result,
	})
	require.NoError(t, err)
	require.Equal(t, domain.SubscriptionStatusPastDue, updated.Status)
	require.Equal(t, 1, updated.Retries)

	return updated
}

// seatMetric builds a carry-over seat metric for the seat-billing aggregations.
func seatMetric(orgId string, agg domain.AggregationType, fieldName string) domain.BillableMetric {
	return domain.BillableMetric{
		OrgId: orgId, Code: "seats", Aggregation: agg, FieldName: fieldName,
		CarryOver: true, RoundingMode: "round", RoundingScale: 2,
	}
}

// ---------------------------------------------------------------------------
// service builders
// ---------------------------------------------------------------------------

// buildSubscriptionService mirrors app.go's NewSubscriptionService wiring, but
// with the memory gateway registered in the GatewayFactory and no-op pubsub /
// error reporter. Repos are constructed straight off the testcontainer pool.
func buildSubscriptionService(t *testing.T, pool *pgxpool.Pool) *service.SubscriptionService {
	t.Helper()

	logger := noopLogger{}
	pubsub := noopPubSub{}
	reporter := errors.NewErrorReporter(logger)

	pspRepo := NewPspRepo(pool)
	settingRepo := NewSettingRepo(pool)
	memoryAdapter := memory.NewGatewayAdapter(logger)
	// nil cipher: the memory gateway row stores no credentials, and the
	// factory only opens the cipher when an envelope is present.
	gatewayFactory := service.NewGatewayFactory(
		pspRepo,
		nil,
		logger,
		map[domain.Gateway]port.GatewayAdapter{domain.Memory: memoryAdapter},
	)

	// Invoice-centric billing (Spec A): the charge amount comes from a per-cycle
	// invoice. Mirror app.go's narrow-service wiring.
	usageEventStore := NewEventStore(pool)
	usageService := service.NewUsageService(NewMeterRepo(pool), NewCustomerRepo(pool), NewSubscriptionRepo(pool), NewOrderRepo(pool), NewPriceRepo(pool), usageEventStore, usageEventStore, pubsub, logger)
	invoiceService := service.NewInvoiceService(NewInvoiceRepo(pool), NewOrderRepo(pool), NewPriceRepo(pool), NewSubscriptionRepo(pool), usageService, NewTxManager(pool), logger, NewDiscountRepo(pool), NewCouponRepo(pool), NewCouponReservationRepo(pool), service.NewInvoiceSettingsService(NewSettingRepo(pool), logger))

	svc, err := service.NewSubscriptionService(
		NewSessionRepo(pool),
		settingRepo,
		NewCartRepo(pool),
		NewSubscriptionRepo(pool),
		NewCustomerRepo(pool),
		NewOrderRepo(pool),
		NewPaymentRepo(pool),
		NewPriceRepo(pool),
		gatewayFactory,
		invoiceService,
		pubsub,
		reporter,
		logger,
		NewTxManager(pool),
	)
	require.NoError(t, err)
	return svc
}

// buildInvoiceService mirrors buildSubscriptionService's invoice wiring for
// tests that drive the invoice builder directly.
func buildInvoiceService(t *testing.T, pool *pgxpool.Pool) *service.InvoiceService {
	t.Helper()
	return service.NewInvoiceService(NewInvoiceRepo(pool), NewOrderRepo(pool), NewPriceRepo(pool), NewSubscriptionRepo(pool),
		buildUsageService(t, pool), NewTxManager(pool), noopLogger{}, NewDiscountRepo(pool), NewCouponRepo(pool), NewCouponReservationRepo(pool),
		service.NewInvoiceSettingsService(NewSettingRepo(pool), noopLogger{}))
}

// buildWiredInvoiceService constructs the real InvoiceService off the
// testcontainer pool, with a real InvoiceSettingsService resolver (mirroring
// app.go's construction order: settings → invoice). This is the service the
// order flow drives end to end.
func buildWiredInvoiceService(t *testing.T, pool *pgxpool.Pool) *service.InvoiceService {
	t.Helper()
	logger := noopLogger{}
	settings := service.NewInvoiceSettingsService(NewSettingRepo(pool), logger)
	return service.NewInvoiceService(
		NewInvoiceRepo(pool),
		NewOrderRepo(pool),
		NewPriceRepo(pool),
		NewSubscriptionRepo(pool),
		buildUsageService(t, pool),
		NewTxManager(pool),
		logger,
		NewDiscountRepo(pool),
		NewCouponRepo(pool),
		NewCouponReservationRepo(pool),
		settings, // real resolver
	)
}

// buildUsageService wires a UsageService off the testcontainer pool, with the
// EventStore as both the durable ingestor and the read/aggregation backend (the
// USAGE_DATABASE_URL-unset, sync-ingest production default).
func buildUsageService(t *testing.T, pool *pgxpool.Pool) *service.UsageService {
	t.Helper()
	store := NewEventStore(pool)
	return service.NewUsageService(
		NewMeterRepo(pool),
		NewCustomerRepo(pool),
		NewSubscriptionRepo(pool),
		NewOrderRepo(pool),
		NewPriceRepo(pool),
		store, // ingestor
		store, // event store
		noopPubSub{},
		noopLogger{},
	)
}

// buildCouponService mirrors app.go's NewCouponService wiring off the
// testcontainer pool (real reservation/discount/code repos so the reserve →
// consume → discount path is exercised end to end).
func buildCouponService(t *testing.T, pool *pgxpool.Pool) *service.CouponService {
	t.Helper()
	return service.NewCouponService(
		NewCouponRepo(pool),
		NewCouponCodeRepo(pool),
		NewDiscountRepo(pool),
		noopPriorPayments{},
		NewTxManager(pool),
		noopLogger{},
		NewCouponReservationRepo(pool),
	)
}

// buildOrderService wires an engine-aware OrderService off the testcontainer pool,
// with the memory gateway registered and the coupon service threaded in.
func buildOrderService(t *testing.T, pool *pgxpool.Pool, coupons *service.CouponService) *service.OrderService {
	t.Helper()
	logger := noopLogger{}
	pspRepo := NewPspRepo(pool)
	gatewayFactory := service.NewGatewayFactory(
		pspRepo,
		nil,
		logger,
		map[domain.Gateway]port.GatewayAdapter{domain.Memory: memory.NewGatewayAdapter(logger)},
	)
	return service.NewOrderService(
		NewTxManager(pool),
		noopEngine{},
		NewSessionRepo(pool),
		NewPriceRepo(pool),
		NewCartRepo(pool),
		NewOrderRepo(pool),
		NewCustomerRepo(pool),
		NewSubscriptionRepo(pool),
		NewPaymentRepo(pool),
		NewPaymentMethodRepo(pool),
		NewProductRepo(pool),
		gatewayFactory,
		noopPubSub{},
		logger,
		coupons,
		noopOrderInvoicing{}, // this flow does not opt into order-level invoicing
	)
}

// buildWiredOrderService wires an engine-aware OrderService off the testcontainer
// pool with the memory gateway, the coupon service, and a REAL InvoiceService —
// mirroring app.go's NewOrderService argument order. Unlike buildOrderService
// (which passes noop invoicing), this exercises the full order-invoicing path.
func buildWiredOrderService(t *testing.T, pool *pgxpool.Pool, coupons *service.CouponService, invoices *service.InvoiceService) *service.OrderService {
	t.Helper()
	logger := noopLogger{}
	gatewayFactory := service.NewGatewayFactory(
		NewPspRepo(pool),
		nil,
		logger,
		map[domain.Gateway]port.GatewayAdapter{domain.Memory: memory.NewGatewayAdapter(logger)},
	)
	return service.NewOrderService(
		NewTxManager(pool),
		noopEngine{},
		NewSessionRepo(pool),
		NewPriceRepo(pool),
		NewCartRepo(pool),
		NewOrderRepo(pool),
		NewCustomerRepo(pool),
		NewSubscriptionRepo(pool),
		NewPaymentRepo(pool),
		NewPaymentMethodRepo(pool),
		NewProductRepo(pool),
		gatewayFactory,
		noopPubSub{},
		logger,
		coupons,
		invoices,
	)
}

// buildWiredOrderWorkflowService wires the webhook-path OrderWorkflowService off
// the testcontainer pool with a REAL InvoiceService + CouponService — mirroring
// app.go's NewOrderWorkflowService argument order. This is the PSP-webhook
// completion path (CompleteCheckoutSession).
func buildWiredOrderWorkflowService(t *testing.T, pool *pgxpool.Pool, coupons *service.CouponService, invoices *service.InvoiceService) *service.OrderWorkflowService {
	t.Helper()
	logger := noopLogger{}
	return service.NewOrderWorkflowService(
		NewOrderRepo(pool),
		NewCustomerRepo(pool),
		NewSubscriptionRepo(pool),
		NewPaymentMethodRepo(pool),
		NewPaymentRepo(pool),
		NewPriceRepo(pool),
		NewTxManager(pool),
		noopPubSub{},
		logger,
		invoices,
		coupons,
	)
}

// ---------------------------------------------------------------------------
// usage recording helpers
// ---------------------------------------------------------------------------

// recordUsage drives a usage event through the full RecordEvent validation +
// ingest path (the same path the HTTP handler uses) and asserts the outcome
// status, returning it for further assertions.
func recordUsage(t *testing.T, usage *service.UsageService, in port.RecordEventInput) port.IngestResult {
	t.Helper()
	res, err := usage.RecordEvent(context.Background(), in)
	require.NoError(t, err)
	return res
}

// recordEmails records one batch-send usage event of `emails` emails inside the
// period, through the full RecordEvent validation + ingest path.
func recordEmails(t *testing.T, usage *service.UsageService, orgId string, fx graduatedEmailFixture, extId string, emails int, ts time.Time) {
	t.Helper()
	_, err := usage.RecordEvent(context.Background(), port.RecordEventInput{
		OrgId:          orgId,
		CustomerId:     fx.customer.Id,
		MetricCode:     fx.meter.Code,
		SubscriptionId: fx.sub.Id,
		ExternalId:     extId,
		Metadata:       map[string]string{"emails": decimal.NewFromInt(int64(emails)).String()},
		Timestamp:      ts,
	})
	require.NoError(t, err)
}

// addRemove records one add/remove event through the full RecordEvent path.
func addRemove(t *testing.T, pool *pgxpool.Pool, fx meteredFixture, orgId, extId, op, identity string, ts time.Time) {
	t.Helper()
	usage := buildUsageService(t, pool)
	recordUsage(t, usage, port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: extId, Timestamp: ts,
		Metadata: map[string]string{domain.UsageOperationKey: op, fx.meter.FieldName: identity},
	})
}

// levelReport records one level report through the full RecordEvent path.
func levelReport(t *testing.T, pool *pgxpool.Pool, fx meteredFixture, orgId, extId, value string, ts time.Time) {
	t.Helper()
	usage := buildUsageService(t, pool)
	recordUsage(t, usage, port.RecordEventInput{
		OrgId: orgId, CustomerId: fx.customer.Id, MetricCode: fx.meter.Code,
		SubscriptionId: fx.sub.Id, ExternalId: extId, Timestamp: ts,
		Metadata: map[string]string{fx.meter.FieldName: value},
	})
}

// chargeAndAssertInvoice runs the real charge for the fixture's current period and
// asserts the charge amount, then the persisted invoice's single usage line:
// quantity and line total, and the invoice total.
func chargeAndAssertInvoice(t *testing.T, pool *pgxpool.Pool, orgId string, fx meteredFixture, wantQty string, wantTotalCents int64) {
	t.Helper()
	ctx := context.Background()

	svc := buildSubscriptionService(t, pool)
	result, err := svc.ChargeForBillingPeriod(ctx, fx.sub)
	require.NoError(t, err)
	assert.Equal(t, wantTotalCents, result.Amount, "charge amount")

	inv, err := NewInvoiceRepo(pool).FindBySubscriptionCycle(ctx, orgId, fx.sub.Id, 0)
	require.NoError(t, err, "an invoice must exist for the billed cycle")
	assert.Equal(t, wantTotalCents, inv.Total, "invoice total")
	require.Len(t, inv.LineItems, 1, "single metered item → one usage line")
	line := inv.LineItems[0]
	assert.Equal(t, domain.InvoiceLineKindUsage, line.Kind)
	assert.True(t, line.Quantity.Equal(decimal.RequireFromString(wantQty)),
		"usage line quantity: got %s want %s", line.Quantity, wantQty)
	assert.Equal(t, wantTotalCents, line.Total, "usage line total")
}

// ---------------------------------------------------------------------------
// jetstream helper (usage ingest e2e)
// ---------------------------------------------------------------------------

// embeddedJS starts an in-process JetStream server for the test.
func embeddedJS(t *testing.T) jetstream.JetStream {
	t.Helper()
	ns, err := natsserver.NewServer(&natsserver.Options{
		ServerName: "e2e_js", DontListen: true, JetStream: true, StoreDir: t.TempDir(),
	})
	require.NoError(t, err)
	go ns.Start()
	require.True(t, ns.ReadyForConnections(5*time.Second), "embedded nats not ready")
	nc, err := nats.Connect("", nats.InProcessServer(ns))
	require.NoError(t, err)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	t.Cleanup(func() { nc.Close(); ns.Shutdown() })
	return js
}
