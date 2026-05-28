//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// subFixture seeds the parent chain (customer, price, order, order item) the
// subscription's foreign keys point at, and returns a ready-to-create
// Subscription wired to them.
type subFixture struct {
	customer domain.Customer
	order    domain.Order
	item     domain.OrderItem
	sub      domain.Subscription
}

func seedSubFixture(t *testing.T, db *gorm.DB, orgId string) subFixture {
	t.Helper()
	cust := seedCustomer(t, db, orgId)
	price := seedPrice(t, db, orgId)
	order := seedOrder(t, db, orgId, cust.Id)
	item := seedOrderItem(t, db, orgId, order.Id, price.Id)
	return subFixture{customer: cust, order: order, item: item, sub: newSubscription(orgId, cust.Id, order.Id, item.Id)}
}

func newSubscription(orgId, customerId, orderId, orderItemId string) domain.Subscription {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return domain.Subscription{
		OrgId:              orgId,
		Id:                 lib.GenerateId("sub"),
		PspId:              domain.Paystack,
		OrderId:            orderId,
		OrderItemId:        orderItemId,
		CustomerId:         customerId,
		Status:             domain.SubscriptionStatusActive,
		StartDate:          now,
		BillingInterval:    domain.BillingIntervalMonth,
		BillingIntervalQty: 1,
		Cycles:             12,
		Currency:           "USD",
		Amount:             1999,
		Metadata:           map[string]string{"plan": "pro"},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func TestSubscriptionRepo(t *testing.T) {
	db := testDB(t)
	repo := NewSubscriptionRepo(db)
	ctx := context.Background()

	t.Run("Create then FindById round-trips", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		fx := seedSubFixture(t, db, orgId)
		sub := fx.sub

		created, err := repo.Create(ctx, sub)
		require.NoError(t, err)
		assert.Equal(t, sub.Id, created.Id)
		assert.Equal(t, sub.Amount, created.Amount)
		assert.Equal(t, domain.SubscriptionStatusActive, created.Status)
		// Customer is preloaded on the create round-trip.
		assert.Equal(t, fx.customer.Id, created.Customer.Id)
		assert.Equal(t, fx.customer.Email, created.Customer.Email)
		assert.Equal(t, map[string]string{"plan": "pro"}, created.Metadata)

		got, err := repo.FindById(ctx, orgId, sub.Id)
		require.NoError(t, err)
		assert.Equal(t, created.Id, got.Id)
		assert.Equal(t, created.Amount, got.Amount)
		assert.Equal(t, fx.customer.Id, got.Customer.Id)
	})

	t.Run("Update mutates fields", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		fx := seedSubFixture(t, db, orgId)
		created, err := repo.Create(ctx, fx.sub)
		require.NoError(t, err)

		created.Status = domain.SubscriptionStatusCancelled
		created.Amount = 4999
		created.CancelledAt = time.Now().UTC().Truncate(time.Microsecond)

		updated, err := repo.Update(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusCancelled, updated.Status)
		assert.Equal(t, int64(4999), updated.Amount)

		reread, err := repo.FindById(ctx, orgId, created.Id)
		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusCancelled, reread.Status)
		assert.Equal(t, int64(4999), reread.Amount)
	})

	t.Run("FindById not-found returns ErrRecordNotFound", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		_, err := repo.FindById(ctx, orgId, "does-not-exist")
		assert.True(t, errors.Is(err, port.ErrNotFound), "want ErrRecordNotFound, got %v", err)
	})

	t.Run("FindByOrderId returns only matching subs", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		fx := seedSubFixture(t, db, orgId)
		orderId := fx.order.Id

		s1 := newSubscription(orgId, fx.customer.Id, orderId, fx.item.Id)
		_, err := repo.Create(ctx, s1)
		require.NoError(t, err)

		s2 := newSubscription(orgId, fx.customer.Id, orderId, fx.item.Id)
		_, err = repo.Create(ctx, s2)
		require.NoError(t, err)

		// A third sub on a different order must be excluded.
		otherOrder := seedOrder(t, db, orgId, fx.customer.Id)
		otherItem := seedOrderItem(t, db, orgId, otherOrder.Id, seedPrice(t, db, orgId).Id)
		other := newSubscription(orgId, fx.customer.Id, otherOrder.Id, otherItem.Id)
		_, err = repo.Create(ctx, other)
		require.NoError(t, err)

		got, err := repo.FindByOrderId(ctx, orgId, orderId)
		require.NoError(t, err)
		assert.Len(t, got, 2)
		for _, s := range got {
			assert.Equal(t, orderId, s.OrderId)
		}
	})

	t.Run("Find paginates and counts within org", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		fx := seedSubFixture(t, db, orgId)
		for range 3 {
			_, err := repo.Create(ctx, newSubscription(orgId, fx.customer.Id, fx.order.Id, fx.item.Id))
			require.NoError(t, err)
		}

		p := domain.Pagination{Limit: 2, Offset: 0, SortBy: "created_at", SortDirection: "asc"}
		subs, count, err := repo.Find(ctx, orgId, p)
		require.NoError(t, err)
		assert.Equal(t, 3, count, "total count ignores pagination limit")
		assert.Len(t, subs, 2, "page is limited to 2")
	})

	t.Run("org-scoping isolates rows", func(t *testing.T) {
		orgA := uniqueOrg(t)
		orgB := uniqueOrg(t)
		cleanupOrg(t, db, orgA)
		cleanupOrg(t, db, orgB)
		fxA := seedSubFixture(t, db, orgA)
		created, err := repo.Create(ctx, fxA.sub)
		require.NoError(t, err)

		// Same id, queried under org B -> not found.
		_, err = repo.FindById(ctx, orgB, created.Id)
		assert.True(t, errors.Is(err, port.ErrNotFound), "row in org A must not be visible to org B")

		// org B Find returns nothing.
		subs, count, err := repo.Find(ctx, orgB, domain.Pagination{Limit: 10, SortBy: "created_at", SortDirection: "asc"})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Empty(t, subs)
	})
}
