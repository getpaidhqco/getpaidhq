package storagetest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

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
	t.Run("TxRollback", func(t *testing.T) { testTxRollback(t, ctx, rs) })
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
