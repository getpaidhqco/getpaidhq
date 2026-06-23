package storagetest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// RepoSet is the set of storage ports the conformance suite exercises. Each
// adapter's integration test builds one (from its own pool/handle) and hands it
// to RunConformance, so the SAME assertions run against every driver — the
// parity guarantee. Fields are added as the suite grows.
type RepoSet struct {
	Org          port.OrgRepository
	Customer     port.CustomerRepository
	Product      port.ProductRepository
	Variant      port.VariantRepository
	Price        port.PriceRepository
	Cart         port.CartRepository
	Order        port.OrderRepository
	Subscription port.SubscriptionRepository
	Payment      port.PaymentRepository
	Setting      port.SettingRepository
	Idempotency  port.IdempotencyKeyRepository
	Tx           port.TxManager

	IdempotencyStore port.IdempotencyStore

	Invoice           port.InvoiceRepository
	Dunning           port.DunningRepository
	Coupon            port.CouponRepository
	CouponCode        port.CouponCodeRepository
	CouponReservation port.CouponReservationRepository
	Discount          port.DiscountRepository
	Meter             port.MeterRepository
	Metadata          port.MetadataStoreRepository
	Psp               port.PspRepository
	ApiKey            port.ApiKeyRepository
	Webhook           port.WebhookSubscriptionRepository
	Session           port.SessionRepository
	PaymentMethod     port.PaymentMethodRepository
	EventStore        port.EventStore
}

// Factory builds a RepoSet bound to a fresh connection against dsn. Defined in
// each adapter's test package so it can import that adapter.
type Factory func(t *testing.T, dsn string) RepoSet

// RunConformance boots the shared container and runs the cross-driver suite
// against the RepoSet the factory builds. Each sub-test uses a unique org id so
// rows never collide across tests sharing the container.
func RunConformance(t *testing.T, newRepos Factory) {
	dsn := StartPostgres(t)
	rs := newRepos(t, dsn)
	ctx := context.Background()

	t.Run("OrgAndCustomer", func(t *testing.T) { testOrgAndCustomer(t, ctx, rs) })
	t.Run("ProductVariantPrice", func(t *testing.T) { testProductVariantPrice(t, ctx, rs) })
	t.Run("CartOrderItem", func(t *testing.T) { testCartOrderItem(t, ctx, rs) })
	t.Run("Subscription", func(t *testing.T) { testSubscription(t, ctx, rs) })
	t.Run("Payment", func(t *testing.T) { testPayment(t, ctx, rs) })
	t.Run("SettingUpsert", func(t *testing.T) { testSettingUpsert(t, ctx, rs) })
	t.Run("IdempotencyClaim", func(t *testing.T) { testIdempotency(t, ctx, rs) })
	t.Run("IdempotencyStore", func(t *testing.T) { testIdempotencyStore(t, ctx, rs) })
	t.Run("TxRollback", func(t *testing.T) { testTxRollback(t, ctx, rs) })
	t.Run("ProductFindAndVariantDelete", func(t *testing.T) { testProductFindAndVariantDelete(t, ctx, rs) })
	t.Run("Invoice", func(t *testing.T) { testInvoice(t, ctx, rs) })
	t.Run("Dunning", func(t *testing.T) { testDunning(t, ctx, rs) })
	t.Run("Coupon", func(t *testing.T) { testCoupon(t, ctx, rs) })
	t.Run("CouponReservation", func(t *testing.T) { testCouponReservation(t, ctx, rs) })
	t.Run("Meter", func(t *testing.T) { testMeter(t, ctx, rs) })
	t.Run("Metadata", func(t *testing.T) { testMetadata(t, ctx, rs) })
	t.Run("Psp", func(t *testing.T) { testPsp(t, ctx, rs) })
	t.Run("ApiKey", func(t *testing.T) { testApiKey(t, ctx, rs) })
	t.Run("Webhook", func(t *testing.T) { testWebhook(t, ctx, rs) })
	t.Run("Session", func(t *testing.T) { testSession(t, ctx, rs) })
	t.Run("PaymentMethod", func(t *testing.T) { testPaymentMethod(t, ctx, rs) })
	t.Run("EventStore", func(t *testing.T) { testEventStore(t, ctx, rs) })
}

func now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// ---- ports-based seeders (driver-agnostic; mirror the gorm seed helpers) ----

func seedOrg(t *testing.T, ctx context.Context, rs RepoSet) string {
	t.Helper()
	orgId := lib.GenerateId("org_test")
	_, err := rs.Org.Create(ctx, domain.Org{
		Id: orgId, Name: "Test Org " + orgId, Country: "US", Timezone: "UTC",
		Status: domain.OrgStatusActive, CreatedAt: now(), UpdatedAt: now(),
	})
	require.NoError(t, err)
	return orgId
}

func seedCustomer(t *testing.T, ctx context.Context, rs RepoSet, orgId string) domain.Customer {
	t.Helper()
	c := domain.Customer{
		OrgId: orgId, Id: lib.GenerateId("cus"),
		FirstName: "Ada", LastName: "Lovelace",
		Email:          fmt.Sprintf("%s@example.com", lib.GenerateId("ada")),
		Phone:          "+15551234",
		BillingAddress: domain.Address{Line1: "1 Engine Way", City: "London", Country: "GB"},
		Metadata:       map[string]string{"tier": "gold"},
		CreatedAt:      now(), UpdatedAt: now(),
	}
	created, err := rs.Customer.Create(ctx, c)
	require.NoError(t, err)
	return created
}

func seedPrice(t *testing.T, ctx context.Context, rs RepoSet, orgId string) domain.Price {
	t.Helper()
	prod := domain.Product{OrgId: orgId, Id: lib.GenerateId("prod"), Name: "Test Product", Status: domain.ProductStatusActive, CreatedAt: now(), UpdatedAt: now()}
	_, err := rs.Product.Create(ctx, prod)
	require.NoError(t, err)
	variant := domain.Variant{OrgId: orgId, Id: lib.GenerateId("var"), ProductId: prod.Id, Name: "Default", CreatedAt: now(), UpdatedAt: now()}
	_, err = rs.Variant.Create(ctx, variant)
	require.NoError(t, err)
	price := domain.Price{
		OrgId: orgId, Id: lib.GenerateId("price"), VariantId: variant.Id,
		Label: "Monthly Pro", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed,
		Currency: domain.USD, UnitPrice: 1999, BillingInterval: domain.BillingIntervalMonth,
		BillingIntervalQty: 1, TrialInterval: domain.BillingIntervalNone, CreatedAt: now(), UpdatedAt: now(),
	}
	created, err := rs.Price.Create(ctx, price)
	require.NoError(t, err)
	return created
}

func seedOrder(t *testing.T, ctx context.Context, rs RepoSet, orgId, customerId string) domain.Order {
	t.Helper()
	cart := domain.Cart{OrgId: orgId, Id: lib.GenerateId("cart"), Metadata: map[string]string{}, CreatedAt: now(), UpdatedAt: now()}
	_, err := rs.Cart.Create(ctx, cart)
	require.NoError(t, err)
	o := domain.Order{
		OrgId: orgId, Id: lib.GenerateId("ord"), CustomerId: customerId, CartId: cart.Id,
		Reference: "REF-" + lib.GenerateId("r"), Status: domain.OrderStatusPending, Currency: "USD",
		Total: 1999, Metadata: map[string]string{}, CreatedAt: now(), UpdatedAt: now(),
	}
	created, err := rs.Order.Create(ctx, o)
	require.NoError(t, err)
	return created
}

// ---- sub-tests ----

func testOrgAndCustomer(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	c := seedCustomer(t, ctx, rs, orgId)

	got, err := rs.Customer.FindById(ctx, orgId, c.Id)
	require.NoError(t, err)
	assert.Equal(t, c.Email, got.Email)
	assert.Equal(t, "London", got.BillingAddress.City)
	assert.Equal(t, "gold", got.Metadata["tier"])
	assert.Equal(t, "", got.ExternalId, "unset external_id round-trips as empty")

	byEmail, err := rs.Customer.FindByEmail(ctx, orgId, c.Email)
	require.NoError(t, err)
	assert.Equal(t, c.Id, byEmail.Id)

	got.LastName = "Byron"
	updated, err := rs.Customer.Update(ctx, got)
	require.NoError(t, err)
	assert.Equal(t, "Byron", updated.LastName)

	_, total, err := rs.Customer.List(ctx, orgId, domain.Pagination{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)

	_, err = rs.Customer.FindById(ctx, orgId, "nope")
	assert.ErrorIs(t, err, port.ErrNotFound)
}

func testProductVariantPrice(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	price := seedPrice(t, ctx, rs, orgId)

	got, err := rs.Price.FindById(ctx, orgId, price.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(1999), got.UnitPrice)
	assert.Equal(t, domain.BillingIntervalMonth, got.BillingInterval)

	byVariant, _, err := rs.Price.FindByVariantId(ctx, orgId, price.VariantId, domain.Pagination{Limit: 10})
	require.NoError(t, err)
	require.Len(t, byVariant, 1)
	assert.Equal(t, price.Id, byVariant[0].Id)

	batch, err := rs.Price.FindByIds(ctx, orgId, []string{price.Id})
	require.NoError(t, err)
	require.Len(t, batch, 1)
}

func testCartOrderItem(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	cust := seedCustomer(t, ctx, rs, orgId)
	price := seedPrice(t, ctx, rs, orgId)
	order := seedOrder(t, ctx, rs, orgId, cust.Id)

	item := domain.OrderItem{
		OrgId: orgId, Id: lib.GenerateId("oi"), OrderId: order.Id, PriceId: price.Id,
		ProductId: "test-product", Description: "Monthly Pro", Quantity: 1,
		Subtotal: 1999, Total: 1999, Metadata: map[string]string{}, CreatedAt: now(), UpdatedAt: now(),
	}
	_, err := rs.Order.CreateOrderItem(ctx, item)
	require.NoError(t, err)

	got, err := rs.Order.FindById(ctx, orgId, order.Id)
	require.NoError(t, err)
	assert.Equal(t, order.Id, got.Id)

	err = rs.Tx.RunInTx(ctx, func(ctx context.Context) error {
		locked, err := rs.Order.FindByIdForUpdate(ctx, orgId, order.Id)
		if err != nil {
			return err
		}
		assert.Equal(t, order.Id, locked.Id)
		return nil
	})
	require.NoError(t, err)

	err = rs.Tx.RunInTx(ctx, func(ctx context.Context) error {
		_, err := rs.Order.FindByIdForUpdate(ctx, orgId, "missing_order")
		return err
	})
	assert.ErrorIs(t, err, port.ErrNotFound)

	items, err := rs.Order.FindOrderItemsByOrderId(ctx, orgId, order.Id)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, price.Id, items[0].PriceId)
}

func newSubscription(orgId, customerId, orderId string) domain.Subscription {
	n := now()
	return domain.Subscription{
		OrgId: orgId, Id: lib.GenerateId("sub"), PspId: domain.Paystack, OrderId: orderId,
		CustomerId: customerId, Status: domain.SubscriptionStatusActive, StartDate: n,
		RenewsAt: n.Add(-time.Hour), BillingInterval: domain.BillingIntervalMonth, BillingIntervalQty: 1,
		TrialInterval: domain.BillingIntervalNone, Cycles: 12, Currency: "USD",
		Metadata: map[string]string{"plan": "pro"}, CreatedAt: n, UpdatedAt: n,
	}
}

func testSubscription(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	cust := seedCustomer(t, ctx, rs, orgId)
	price := seedPrice(t, ctx, rs, orgId)
	order := seedOrder(t, ctx, rs, orgId, cust.Id)
	item := domain.OrderItem{
		OrgId: orgId, Id: lib.GenerateId("oi"), OrderId: order.Id, PriceId: price.Id,
		ProductId: "test-product", Description: "Monthly Pro", Quantity: 1,
		Subtotal: 1999, Total: 1999, Metadata: map[string]string{}, CreatedAt: now(), UpdatedAt: now(),
	}
	_, err := rs.Order.CreateOrderItem(ctx, item)
	require.NoError(t, err)

	sub := newSubscription(orgId, cust.Id, order.Id)
	created, err := rs.Subscription.Create(ctx, sub)
	require.NoError(t, err)
	assert.Equal(t, domain.SubscriptionStatusActive, created.Status)
	assert.Equal(t, map[string]string{"plan": "pro"}, created.Metadata)

	got, err := rs.Subscription.FindById(ctx, orgId, sub.Id)
	require.NoError(t, err)
	assert.Equal(t, sub.Id, got.Id)

	// FindByIdForUpdate must work inside a transaction (row lock).
	err = rs.Tx.RunInTx(ctx, func(ctx context.Context) error {
		locked, err := rs.Subscription.FindByIdForUpdate(ctx, orgId, sub.Id)
		if err != nil {
			return err
		}
		assert.Equal(t, sub.Id, locked.Id)
		return nil
	})
	require.NoError(t, err)

	// RenewsAt is in the past + status active → due for billing.
	due, err := rs.Subscription.FindDueForBilling(ctx, orgId, now())
	require.NoError(t, err)
	ids := make([]string, len(due))
	for i, d := range due {
		ids[i] = d.Id
	}
	assert.Contains(t, ids, sub.Id, "active sub with past renews_at is due")
}

func newPayment(orgId, orderId string) domain.Payment {
	n := now()
	return domain.Payment{
		OrgId: orgId, Id: lib.GenerateId("pay"), OrderId: orderId, Psp: domain.Paystack,
		PspId: lib.GenerateId("psp"), Reference: "REF-" + lib.GenerateId("r"),
		Status: domain.PaymentStatusSucceeded, Recurring: true, Currency: "USD",
		Amount: 1999, PspFee: 59, NetAmount: 1940, Metadata: map[string]string{"channel": "card"},
		CompletedAt: n, CreatedAt: n, UpdatedAt: n,
	}
}

func testPayment(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	cust := seedCustomer(t, ctx, rs, orgId)
	order := seedOrder(t, ctx, rs, orgId, cust.Id)

	p := newPayment(orgId, order.Id)
	created, err := rs.Payment.Create(ctx, p)
	require.NoError(t, err)
	assert.Equal(t, int64(1999), created.Amount)
	assert.Equal(t, domain.PaymentStatusSucceeded, created.Status)

	got, err := rs.Payment.FindById(ctx, orgId, p.Id)
	require.NoError(t, err)
	assert.Equal(t, p.Id, got.Id)

	byPsp, err := rs.Payment.FindByPspId(ctx, orgId, p.PspId)
	require.NoError(t, err)
	assert.Equal(t, p.Id, byPsp.Id)
}

func testSettingUpsert(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	s := domain.Setting{OrgId: orgId, ParentId: "", Id: "theme", Type: "string", Value: "dark", CreatedAt: now(), UpdatedAt: now()}
	_, err := rs.Setting.Upsert(ctx, s)
	require.NoError(t, err)

	got, err := rs.Setting.FindById(ctx, orgId, "", "theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", got.Value)

	s.Value = "light"
	_, err = rs.Setting.Upsert(ctx, s)
	require.NoError(t, err)
	got, err = rs.Setting.FindById(ctx, orgId, "", "theme")
	require.NoError(t, err)
	assert.Equal(t, "light", got.Value, "upsert replaces the value on conflict")
}

func testIdempotency(t *testing.T, ctx context.Context, rs RepoSet) {
	key := lib.GenerateId("idem")
	exp := now().Add(time.Hour)

	claimed, err := rs.Idempotency.Claim(ctx, key, exp)
	require.NoError(t, err)
	assert.True(t, claimed, "first claim wins")

	claimed, err = rs.Idempotency.Claim(ctx, key, exp)
	require.NoError(t, err)
	assert.False(t, claimed, "second claim of a live key loses")

	require.NoError(t, rs.Idempotency.Release(ctx, key))

	claimed, err = rs.Idempotency.Claim(ctx, key, exp)
	require.NoError(t, err)
	assert.True(t, claimed, "claim after release wins again")
}

func testTxRollback(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	cust := domain.Customer{
		OrgId: orgId, Id: lib.GenerateId("cus"), Email: fmt.Sprintf("%s@x.com", lib.GenerateId("e")),
		Metadata: map[string]string{}, CreatedAt: now(), UpdatedAt: now(),
	}
	boom := errors.New("boom")
	err := rs.Tx.RunInTx(ctx, func(ctx context.Context) error {
		if _, err := rs.Customer.Create(ctx, cust); err != nil {
			return err
		}
		return boom // force rollback
	})
	require.ErrorIs(t, err, boom)

	_, err = rs.Customer.FindById(ctx, orgId, cust.Id)
	assert.ErrorIs(t, err, port.ErrNotFound, "customer created inside the rolled-back tx must not persist")
}

// ---- product Find(status) + variant Delete ----

func testProductFindAndVariantDelete(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)

	active := domain.Product{OrgId: orgId, Id: lib.GenerateId("prod"), Name: "Active", Status: domain.ProductStatusActive, CreatedAt: now(), UpdatedAt: now()}
	_, err := rs.Product.Create(ctx, active)
	require.NoError(t, err)
	archived := domain.Product{OrgId: orgId, Id: lib.GenerateId("prod"), Name: "Archived", Status: domain.ProductStatusArchived, CreatedAt: now(), UpdatedAt: now()}
	_, err = rs.Product.Create(ctx, archived)
	require.NoError(t, err)

	// Status filter returns only the active product.
	got, total, err := rs.Product.Find(ctx, orgId, domain.Pagination{Limit: 10}, []domain.ProductStatus{domain.ProductStatusActive})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "status filter excludes archived")
	require.Len(t, got, 1)
	assert.Equal(t, active.Id, got[0].Id)

	// nil statuses returns all.
	_, totalAll, err := rs.Product.Find(ctx, orgId, domain.Pagination{Limit: 10}, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, totalAll, "nil statuses returns all products")

	// Variant Delete: create a variant under active, delete it, confirm gone.
	variant := domain.Variant{OrgId: orgId, Id: lib.GenerateId("var"), ProductId: active.Id, Name: "Default", CreatedAt: now(), UpdatedAt: now()}
	_, err = rs.Variant.Create(ctx, variant)
	require.NoError(t, err)
	_, err = rs.Variant.FindById(ctx, orgId, variant.Id)
	require.NoError(t, err)

	require.NoError(t, rs.Variant.Delete(ctx, orgId, variant.Id))
	_, err = rs.Variant.FindById(ctx, orgId, variant.Id)
	assert.ErrorIs(t, err, port.ErrNotFound, "variant must be gone after Delete")
}

// ---- invoice (atomic create + line items + FindBySubscriptionCycle) ----

func testInvoice(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	cust := seedCustomer(t, ctx, rs, orgId)
	price := seedPrice(t, ctx, rs, orgId)
	subId := lib.GenerateId("sub")

	inv := domain.Invoice{
		OrgId: orgId, Id: lib.GenerateId("inv"), SubscriptionId: subId, CustomerId: cust.Id,
		OrderId: lib.GenerateId("ord"), Status: domain.InvoiceStatusDraft, Currency: "USD",
		Cycle: 3, PeriodStart: now(), PeriodEnd: now().Add(720 * time.Hour),
		Metadata: map[string]string{"reason": "renewal"}, CreatedAt: now(), UpdatedAt: now(),
	}
	inv.AddLine(domain.InvoiceLineItem{
		OrgId: orgId, Id: lib.GenerateId("ili"), InvoiceId: inv.Id, PriceId: price.Id,
		Kind: domain.InvoiceLineKindBase, Description: "Monthly Pro",
		Quantity: decimal.NewFromInt(1), UnitAmount: decimal.NewFromInt(1999), Total: 1999,
		CreatedAt: now(), UpdatedAt: now(),
	})
	inv.AddLine(domain.InvoiceLineItem{
		OrgId: orgId, Id: lib.GenerateId("ili"), InvoiceId: inv.Id, PriceId: price.Id,
		Kind: domain.InvoiceLineKindUsage, Description: "Usage",
		Quantity: decimal.NewFromInt(10), UnitAmount: decimal.NewFromInt(50), Total: 500,
		CreatedAt: now(), UpdatedAt: now(),
	})

	created, err := rs.Invoice.Create(ctx, inv)
	require.NoError(t, err)
	assert.Equal(t, int64(2499), created.Total, "subtotal of both lines")
	require.Len(t, created.LineItems, 2)

	got, err := rs.Invoice.FindById(ctx, orgId, inv.Id)
	require.NoError(t, err)
	assert.Equal(t, inv.Id, got.Id)
	require.Len(t, got.LineItems, 2, "line items persisted atomically with the invoice")
	assert.Equal(t, "renewal", got.Metadata["reason"])

	byCycle, err := rs.Invoice.FindBySubscriptionCycle(ctx, orgId, subId, 3)
	require.NoError(t, err)
	assert.Equal(t, inv.Id, byCycle.Id)

	_, err = rs.Invoice.FindBySubscriptionCycle(ctx, orgId, subId, 99)
	assert.ErrorIs(t, err, port.ErrNotFound, "no invoice for an unbuilt cycle")
}

// ---- dunning (campaign create/find/update + an attempt) ----

func testDunning(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	subId := lib.GenerateId("sub")
	custId := lib.GenerateId("cus")

	campaign := domain.DunningCampaign{
		OrgId: orgId, Id: lib.GenerateId("dun"), SubscriptionId: subId, CustomerId: custId,
		WorkflowId: lib.GenerateId("wf"), Status: domain.DunningStatusActive, FailedAmount: 1999,
		Currency: "USD", InitialFailureReason: "card_declined", StartedAt: now(),
		ConfigSnapshot: map[string]any{"max_attempts": float64(5)},
		Metadata:       map[string]string{"k": "v"}, CreatedAt: now(), UpdatedAt: now(),
	}
	created, err := rs.Dunning.CreateCampaign(ctx, campaign)
	require.NoError(t, err)
	assert.Equal(t, domain.DunningStatusActive, created.Status)
	assert.Equal(t, map[string]any{"max_attempts": float64(5)}, created.ConfigSnapshot)

	got, err := rs.Dunning.FindCampaignById(ctx, orgId, campaign.Id)
	require.NoError(t, err)
	assert.Equal(t, campaign.Id, got.Id)

	active, err := rs.Dunning.FindActiveCampaignForSubscription(ctx, orgId, subId)
	require.NoError(t, err)
	assert.Equal(t, campaign.Id, active.Id)

	got.Status = domain.DunningStatusRecovered
	got.RecoveredAmount = 1999
	updated, err := rs.Dunning.UpdateCampaign(ctx, got)
	require.NoError(t, err)
	assert.Equal(t, domain.DunningStatusRecovered, updated.Status)
	assert.Equal(t, int64(1999), updated.RecoveredAmount)

	attempt := domain.DunningAttempt{
		OrgId: orgId, Id: lib.GenerateId("att"), DunningCampaignId: campaign.Id, SubscriptionId: subId,
		AttemptNumber: 1, AttemptType: domain.DunningAttemptTypeImmediate, Amount: 1999, Currency: "USD",
		Status: domain.PaymentStatusFailed, FailureReason: "insufficient_funds",
		ProcessorResponse: map[string]any{"code": "51"}, AttemptedAt: now(), CreatedAt: now(),
	}
	createdAttempt, err := rs.Dunning.CreateAttempt(ctx, attempt)
	require.NoError(t, err)
	assert.Equal(t, domain.DunningAttemptTypeImmediate, createdAttempt.AttemptType)
	assert.Equal(t, map[string]any{"code": "51"}, createdAttempt.ProcessorResponse)

	attempts, count, err := rs.Dunning.FindAttemptsByCampaignId(ctx, orgId, campaign.Id, domain.Pagination{Limit: 10, SortBy: "attempt_number", SortDirection: "asc"})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, attempts, 1)
	assert.Equal(t, attempt.Id, attempts[0].Id)

	_, err = rs.Dunning.FindCampaignById(ctx, orgId, "missing")
	assert.ErrorIs(t, err, port.ErrNotFound)
}

// ---- coupon + coupon_code + discount ----

func testCoupon(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)

	coupon, err := domain.NewCoupon(domain.NewCouponInput{
		OrgId: orgId, Name: "Quarter", DiscountType: domain.DiscountTypePercentage,
		PercentOff: decimal.NewFromInt(25), Duration: domain.DurationForever,
		AppliesToProducts: []string{"prd_a", "prd_b"}, Metadata: map[string]string{"k": "v"},
	})
	require.NoError(t, err)
	createdCoupon, err := rs.Coupon.Create(ctx, coupon)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(25).Equal(createdCoupon.PercentOff))
	assert.Equal(t, []string{"prd_a", "prd_b"}, createdCoupon.AppliesToProducts)

	gotCoupon, err := rs.Coupon.FindById(ctx, orgId, coupon.Id)
	require.NoError(t, err)
	assert.Equal(t, coupon.Id, gotCoupon.Id)

	updatedCoupon, err := rs.Coupon.UpdateMutable(ctx, orgId, coupon.Id, "Renamed", false, map[string]string{"x": "y"})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updatedCoupon.Name)
	assert.False(t, updatedCoupon.Active)
	assert.True(t, decimal.NewFromInt(25).Equal(updatedCoupon.PercentOff), "terms immutable")

	// Coupon code (case-insensitive lookup + increment redeemed).
	code, err := domain.NewCouponCode(domain.NewCouponCodeInput{
		OrgId: orgId, CouponId: coupon.Id, Code: "summer25",
		Restrictions: domain.Restrictions{FirstTimeTransaction: true},
	})
	require.NoError(t, err)
	_, err = rs.CouponCode.Create(ctx, code)
	require.NoError(t, err)

	byCode, err := rs.CouponCode.FindByCode(ctx, orgId, "SuMmEr25")
	require.NoError(t, err)
	assert.Equal(t, code.Id, byCode.Id)
	assert.True(t, byCode.Restrictions.FirstTimeTransaction)

	require.NoError(t, rs.CouponCode.IncrementRedeemed(ctx, orgId, code.Id))
	byCode, err = rs.CouponCode.FindByCode(ctx, orgId, "SUMMER25")
	require.NoError(t, err)
	assert.Equal(t, 1, byCode.TimesRedeemed)

	// Discount referencing the coupon, scoped to a subscription.
	subId := lib.GenerateId("sub")
	custId := lib.GenerateId("cus")
	disc, err := domain.NewDiscount(domain.NewDiscountInput{
		OrgId: orgId, CouponId: coupon.Id, CustomerId: custId, SubscriptionId: subId,
	})
	require.NoError(t, err)
	_, err = rs.Discount.Create(ctx, disc)
	require.NoError(t, err)

	gotDisc, err := rs.Discount.FindById(ctx, orgId, disc.Id)
	require.NoError(t, err)
	assert.Equal(t, disc.Id, gotDisc.Id)

	forSub, err := rs.Discount.ActiveForSubscription(ctx, orgId, subId)
	require.NoError(t, err)
	require.Len(t, forSub, 1)
	assert.Equal(t, disc.Id, forSub[0].Id)

	byCoupon, err := rs.Discount.CountByCoupon(ctx, orgId, coupon.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, byCoupon)
	byCust, err := rs.Discount.CountByCouponAndCustomer(ctx, orgId, coupon.Id, custId)
	require.NoError(t, err)
	assert.Equal(t, 1, byCust)
}

// ---- coupon_reservation (ephemeral capacity holds) ----

// mustCoupon builds a valid percentage/repeating(2) coupon for orgId.
func mustCoupon(t *testing.T, orgId string) domain.Coupon {
	t.Helper()
	c, err := domain.NewCoupon(domain.NewCouponInput{
		OrgId:            orgId,
		Name:             "Reservation Coupon",
		DiscountType:     domain.DiscountTypePercentage,
		PercentOff:       decimal.NewFromInt(50),
		Duration:         domain.DurationRepeating,
		DurationInCycles: 2,
	})
	require.NoError(t, err)
	return c
}

func testCouponReservation(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)

	coupon := mustCoupon(t, orgId)
	_, err := rs.Coupon.Create(ctx, coupon)
	require.NoError(t, err)

	n := now()
	orderId := lib.GenerateId("ord")
	res, err := domain.NewCouponReservation(domain.NewCouponReservationInput{
		OrgId: orgId, CouponId: coupon.Id, OrderId: orderId,
		ExpiresAt: n.Add(time.Hour),
	})
	require.NoError(t, err)
	created, err := rs.CouponReservation.Create(ctx, res)
	require.NoError(t, err)
	assert.Equal(t, res.Id, created.Id)

	byOrder, err := rs.CouponReservation.FindByOrder(ctx, orgId, orderId)
	require.NoError(t, err)
	require.Len(t, byOrder, 1)
	assert.Equal(t, res.Id, byOrder[0].Id)

	// Live at now; lazily expired at now+2h (no row deletion required).
	live, err := rs.CouponReservation.CountLiveByCoupon(ctx, orgId, coupon.Id, n)
	require.NoError(t, err)
	assert.Equal(t, 1, live, "hold counts while live")

	expired, err := rs.CouponReservation.CountLiveByCoupon(ctx, orgId, coupon.Id, n.Add(2*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, expired, "hold no longer counts past expires_at (lazy expiry)")

	// DeleteByOrder releases the hold.
	require.NoError(t, rs.CouponReservation.DeleteByOrder(ctx, orgId, orderId))
	after, err := rs.CouponReservation.CountLiveByCoupon(ctx, orgId, coupon.Id, n)
	require.NoError(t, err)
	assert.Equal(t, 0, after, "hold gone after DeleteByOrder")
}

// ---- meter (BillableMetric) ----

func testMeter(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)

	m := domain.BillableMetric{
		OrgId: orgId, Id: lib.GenerateId("met"), Code: "api_calls_" + lib.GenerateId("c"),
		Name: "API Calls", Aggregation: domain.AggregationSum, FieldName: "units",
		CarryOver: false, RoundingMode: "round", RoundingScale: 2,
		Filters:  []domain.MetricFilter{{Field: "type", Values: []string{"SMS", "MMS"}}},
		GroupBy:  []string{"project"},
		Metadata: map[string]string{"team": "billing"}, CreatedAt: now(), UpdatedAt: now(),
	}
	created, err := rs.Meter.Create(ctx, m)
	require.NoError(t, err)
	assert.Equal(t, domain.AggregationSum, created.Aggregation)

	byId, err := rs.Meter.FindById(ctx, orgId, m.Id)
	require.NoError(t, err)
	assert.Equal(t, m.Id, byId.Id)
	assert.Equal(t, "units", byId.FieldName)
	require.Len(t, byId.Filters, 1)
	assert.Equal(t, []string{"SMS", "MMS"}, byId.Filters[0].Values)
	assert.Equal(t, []string{"project"}, byId.GroupBy)

	byCode, err := rs.Meter.FindByCode(ctx, orgId, m.Code)
	require.NoError(t, err)
	assert.Equal(t, m.Id, byCode.Id)

	_, total, err := rs.Meter.Find(ctx, orgId, domain.Pagination{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

// ---- metadata_store ----

func testMetadata(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	parentId := lib.GenerateId("ord")

	md := domain.MetadataStore{
		OrgId: orgId, ParentId: parentId, ParentType: "order", Key: "ext_id",
		Value: "shopify_123", Namespace: "integration", CreatedAt: now(), UpdatedAt: now(),
	}
	_, err := rs.Metadata.Create(ctx, md)
	require.NoError(t, err)

	byKey, err := rs.Metadata.FindByKey(ctx, orgId, parentId, "ext_id")
	require.NoError(t, err)
	assert.Equal(t, "shopify_123", byKey.Value)

	byParent, err := rs.Metadata.FindByParent(ctx, orgId, parentId)
	require.NoError(t, err)
	require.Len(t, byParent, 1)
	assert.Equal(t, "ext_id", byParent[0].Key)

	byValue, err := rs.Metadata.FindByValue(ctx, orgId, "ext_id", "shopify_123")
	require.NoError(t, err)
	require.Len(t, byValue, 1)
	assert.Equal(t, parentId, byValue[0].ParentId)

	updated := byKey
	updated.Value = "shopify_456"
	_, err = rs.Metadata.Update(ctx, updated)
	require.NoError(t, err)
	byKey, err = rs.Metadata.FindByKey(ctx, orgId, parentId, "ext_id")
	require.NoError(t, err)
	assert.Equal(t, "shopify_456", byKey.Value)
}

// ---- psp (gateways) ----

func testPsp(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)

	cfg := domain.PspConfig{
		OrgId: orgId, Id: lib.GenerateId("psp"), PspId: domain.Paystack, Name: "Primary",
		Active: true, Config: map[string]string{"channel": "card"},
		EncryptedCredentials: "sealed-envelope", CreatedAt: now(), UpdatedAt: now(),
	}
	created, err := rs.Psp.Create(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, domain.Paystack, created.PspId)

	got, err := rs.Psp.FindById(ctx, orgId, cfg.Id)
	require.NoError(t, err)
	assert.Equal(t, cfg.Id, got.Id)
	assert.True(t, got.Active)
	assert.Equal(t, "card", got.Config["channel"])
	assert.Equal(t, "sealed-envelope", got.EncryptedCredentials)
}

// ---- api_key ----

func testApiKey(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	id := lib.GenerateId("sk")
	key := domain.ApiKey{
		OrgId: orgId, Id: id, Name: "ci-deploy", KeyHash: "hash_" + id,
		CreatedAt: now(), UpdatedAt: now(),
	}
	_, err := rs.ApiKey.Create(ctx, key)
	require.NoError(t, err)

	byId, err := rs.ApiKey.FindById(ctx, orgId, id)
	require.NoError(t, err)
	assert.Equal(t, "ci-deploy", byId.Name)
	assert.Equal(t, key.KeyHash, byId.KeyHash)

	byKey, err := rs.ApiKey.FindByKey(ctx, key.KeyHash)
	require.NoError(t, err)
	assert.Equal(t, id, byKey.Id)
	assert.Equal(t, orgId, byKey.OrgId)

	_, total, err := rs.ApiKey.List(ctx, orgId, domain.Pagination{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)

	require.NoError(t, rs.ApiKey.Delete(ctx, orgId, id))
	_, err = rs.ApiKey.FindById(ctx, orgId, id)
	assert.ErrorIs(t, err, port.ErrNotFound)
}

// ---- webhook_subscription ----

func testWebhook(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	sub := domain.WebhookSubscription{
		OrgID: orgId, Id: lib.GenerateId("whk"), Events: []string{"payment.succeeded", "invoice.paid"},
		URL: "https://example.com/hook", Secret: "whsec_123", CreatedAt: now(), UpdatedAt: now(),
	}
	created, err := rs.Webhook.Create(ctx, sub)
	require.NoError(t, err)
	assert.Equal(t, sub.Id, created.Id)

	got, err := rs.Webhook.GetByID(ctx, orgId, sub.Id)
	require.NoError(t, err)
	assert.Equal(t, []string{"payment.succeeded", "invoice.paid"}, got.Events)
	assert.Equal(t, "https://example.com/hook", got.URL)

	byEvent, err := rs.Webhook.FindByEvent(ctx, orgId, "invoice.paid")
	require.NoError(t, err)
	require.Len(t, byEvent, 1)
	assert.Equal(t, sub.Id, byEvent[0].Id)
}

// ---- session ----

func testSession(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	cart := domain.Cart{OrgId: orgId, Id: lib.GenerateId("cart"), Metadata: map[string]string{}, CreatedAt: now(), UpdatedAt: now()}
	_, err := rs.Cart.Create(ctx, cart)
	require.NoError(t, err)

	s := domain.Session{OrgId: orgId, Id: lib.GenerateId("sess"), CartId: cart.Id, CreatedAt: now(), UpdatedAt: now()}
	created, err := rs.Session.Create(ctx, s)
	require.NoError(t, err)
	assert.Equal(t, s.Id, created.Id)

	got, err := rs.Session.FindById(ctx, orgId, s.Id)
	require.NoError(t, err)
	assert.Equal(t, s.Id, got.Id)
	assert.Equal(t, cart.Id, got.CartId)
}

// ---- payment_method ----

func testPaymentMethod(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	cust := seedCustomer(t, ctx, rs, orgId)

	// One expiring soon, one far in the future.
	expiringAt := now().Add(24 * time.Hour)
	pm := domain.PaymentMethod{
		OrgId: orgId, Id: lib.GenerateId("pm"), Status: domain.PaymentMethodStatusActive,
		Psp: string(domain.Paystack), Name: "Visa", CustomerId: cust.Id,
		BillingAddress: domain.Address{Line1: "1 Card St", City: "London", Country: "GB"},
		Type:           domain.PaymentMethodTypeCard, Token: domain.Secret("authcode_123"),
		Metadata: map[string]string{"brand": "visa"}, ExpireAt: expiringAt, CreatedAt: now(), UpdatedAt: now(),
	}
	created, err := rs.PaymentMethod.Create(ctx, pm)
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentMethodStatusActive, created.Status)

	got, err := rs.PaymentMethod.FindById(ctx, orgId, pm.Id)
	require.NoError(t, err)
	assert.Equal(t, pm.Id, got.Id)
	assert.Equal(t, cust.Id, got.CustomerId)
	assert.Equal(t, "London", got.BillingAddress.City)
	assert.Equal(t, "authcode_123", got.Token.Reveal())

	notExpiring := pm
	notExpiring.Id = lib.GenerateId("pm")
	notExpiring.ExpireAt = now().Add(365 * 24 * time.Hour)
	_, err = rs.PaymentMethod.Create(ctx, notExpiring)
	require.NoError(t, err)

	// FindExpiringPaymentMethods is cross-org; assert our expiring id is in, the far one is out.
	expiring, err := rs.PaymentMethod.FindExpiringPaymentMethods(ctx, expiringAt.Add(time.Hour))
	require.NoError(t, err)
	ids := make(map[string]bool, len(expiring))
	for _, e := range expiring {
		ids[e.Id] = true
	}
	assert.True(t, ids[pm.Id], "the soon-expiring method must be returned")
	assert.False(t, ids[notExpiring.Id], "the far-future method must be excluded")
}

// ---- event_store (Ingest + Count/Sum + ListHistory + AggregateGrouped) ----

func testEventStore(t *testing.T, ctx context.Context, rs RepoSet) {
	orgId := seedOrg(t, ctx, rs)
	custId := lib.GenerateId("cus")
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(12 * time.Hour)

	mk := func(id, ext, sub, project string, val int64, ts time.Time) domain.MeterEvent {
		return domain.MeterEvent{
			OrgId: orgId, Id: id, CustomerId: custId, MetricCode: "api_calls",
			SubscriptionId: sub, ExternalId: ext, Value: decimal.NewFromInt(val),
			Metadata: map[string]string{"project": project}, Timestamp: ts, CreatedAt: ts,
		}
	}
	events := []domain.MeterEvent{
		mk(lib.GenerateId("ev"), lib.GenerateId("x"), "sub_1", "acme", 10, from),
		mk(lib.GenerateId("ev"), lib.GenerateId("x"), "sub_1", "acme", 25, from.Add(2*time.Hour)),
		mk(lib.GenerateId("ev"), lib.GenerateId("x"), "sub_1", "globex", 5, from.Add(3*time.Hour)),
		mk(lib.GenerateId("ev"), lib.GenerateId("x"), "sub_1", "acme", 100, from.Add(24*time.Hour)), // out of window
	}
	for _, e := range events {
		res, err := rs.EventStore.Ingest(ctx, e)
		require.NoError(t, err)
		assert.Equal(t, port.IngestRecorded, res.Status)
	}
	// Resend with a seen external_id must dedup.
	dup := events[0]
	dup.Id = lib.GenerateId("ev")
	res, err := rs.EventStore.Ingest(ctx, dup)
	require.NoError(t, err)
	assert.Equal(t, port.IngestDuplicate, res.Status, "resend with seen external_id is a duplicate")

	q := port.UsageQuery{OrgId: orgId, MetricCode: "api_calls", From: from, To: to, CustomerId: custId, SubscriptionId: "sub_1"}

	n, err := rs.EventStore.Count(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n, "3 in-window attributed (out-of-window excluded)")

	sum, err := rs.EventStore.Sum(ctx, q)
	require.NoError(t, err)
	assert.True(t, sum.Equal(decimal.NewFromInt(40)), "10+25+5, got %s", sum)

	hist, err := rs.EventStore.ListHistory(ctx, q)
	require.NoError(t, err)
	require.Len(t, hist, 3, "ListHistory returns the in-window matches")

	groups, err := rs.EventStore.AggregateGrouped(ctx, q, domain.AggregationSum, "project")
	require.NoError(t, err)
	got := map[string]int64{}
	for _, g := range groups {
		assert.Equal(t, "project", g.Key)
		got[g.Value] = g.Quantity.IntPart()
	}
	assert.Equal(t, map[string]int64{"acme": 35, "globex": 5}, got, "sum split per project")
}

func testIdempotencyStore(t *testing.T, ctx context.Context, rs RepoSet) {
	key := lib.GenerateId("idemreq")
	hashA, hashB := "hash-a", "hash-b"

	c, err := rs.IdempotencyStore.Claim(ctx, key, hashA, "tok-1")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyNew, c.Status)

	c, err = rs.IdempotencyStore.Claim(ctx, key, hashA, "tok-2")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyPending, c.Status)

	// Fencing: Complete with the WRONG token is a no-op (row stays pending).
	require.NoError(t, rs.IdempotencyStore.Complete(ctx, key, "tok-WRONG", 200, []byte(`{"h":1}`), []byte(`{"order":"x"}`)))
	c, err = rs.IdempotencyStore.Claim(ctx, key, hashA, "tok-3")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyPending, c.Status, "wrong-token Complete must not complete the row")

	require.NoError(t, rs.IdempotencyStore.Complete(ctx, key, "tok-1", 201, []byte(`{"h":1}`), []byte(`{"order":"x"}`)))

	c, err = rs.IdempotencyStore.Claim(ctx, key, hashA, "tok-4")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyCompleted, c.Status)
	assert.Equal(t, 201, c.Code)
	assert.Equal(t, []byte(`{"h":1}`), c.Headers)
	assert.Equal(t, []byte(`{"order":"x"}`), c.Body)

	c, err = rs.IdempotencyStore.Claim(ctx, key, hashB, "tok-5")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyConflict, c.Status)

	key2 := lib.GenerateId("idemreq")
	c, err = rs.IdempotencyStore.Claim(ctx, key2, hashA, "tok-a")
	require.NoError(t, err)
	require.Equal(t, port.IdempotencyNew, c.Status)
	require.NoError(t, rs.IdempotencyStore.Abandon(ctx, key2, "tok-WRONG")) // no-op (fenced)
	c, err = rs.IdempotencyStore.Claim(ctx, key2, hashA, "tok-b")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyPending, c.Status, "wrong-token Abandon must not release")
	require.NoError(t, rs.IdempotencyStore.Abandon(ctx, key2, "tok-a")) // real release
	c, err = rs.IdempotencyStore.Claim(ctx, key2, hashA, "tok-c")
	require.NoError(t, err)
	assert.Equal(t, port.IdempotencyNew, c.Status, "claim after release wins again")

	key3 := lib.GenerateId("idemreq")
	const n = 12
	results := make([]port.IdempotencyClaimStatus, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cc, e := rs.IdempotencyStore.Claim(ctx, key3, hashA, fmt.Sprintf("tok-%d", i))
			require.NoError(t, e)
			results[i] = cc.Status
		}(i)
	}
	wg.Wait()
	newCount := 0
	for _, s := range results {
		if s == port.IdempotencyNew {
			newCount++
		}
	}
	assert.Equal(t, 1, newCount, "exactly one concurrent claim wins New")
}
