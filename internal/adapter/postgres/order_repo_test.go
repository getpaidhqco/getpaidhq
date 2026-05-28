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
	"getpaidhq/internal/lib"
)

func TestOrderRepo(t *testing.T) {
	db := testDB(t)
	repo := NewOrderRepo(db)
	ctx := context.Background()

	t.Run("Create then FindById round-trips with customer", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		cust := seedCustomer(t, db, orgId)

		o := domain.Order{
			OrgId:      orgId,
			Id:         lib.GenerateId("ord"),
			CustomerId: cust.Id,
			Reference:  "REF-1",
			Status:     domain.OrderStatusPending,
			Currency:   "USD",
			Total:      4999,
			Metadata:   map[string]string{"source": "web"},
			CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt:  time.Now().UTC().Truncate(time.Microsecond),
		}
		created, err := repo.Create(ctx, o)
		require.NoError(t, err)
		assert.Equal(t, o.Id, created.Id)
		assert.Equal(t, int64(4999), created.Total)
		assert.Equal(t, cust.Id, created.Customer.Id) // preloaded
		assert.Equal(t, map[string]string{"source": "web"}, created.Metadata)

		got, err := repo.FindById(ctx, orgId, o.Id)
		require.NoError(t, err)
		assert.Equal(t, o.Reference, got.Reference)
		assert.Empty(t, got.Items) // no items yet
	})

	t.Run("Update mutates status", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		cust := seedCustomer(t, db, orgId)
		created, err := repo.Create(ctx, seedOrderModel(orgId, cust.Id))
		require.NoError(t, err)

		created.Status = domain.OrderStatusCompleted
		updated, err := repo.Update(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, domain.OrderStatusCompleted, updated.Status)

		reread, err := repo.FindById(ctx, orgId, created.Id)
		require.NoError(t, err)
		assert.Equal(t, domain.OrderStatusCompleted, reread.Status)
	})

	t.Run("FindById not-found returns ErrRecordNotFound", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		_, err := repo.FindById(ctx, orgId, "missing")
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})

	t.Run("order items: create, find, preload price, list by order", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		cust := seedCustomer(t, db, orgId)
		price := seedPrice(t, db, orgId)
		order := seedOrder(t, db, orgId, cust.Id)

		item := domain.OrderItem{
			OrgId:       orgId,
			Id:          lib.GenerateId("oi"),
			OrderId:     order.Id,
			PriceId:     price.Id,
			Description: "Pro plan",
			Quantity:    2,
			Subtotal:    3998,
			Total:       3998,
			Metadata:    map[string]string{"k": "v"},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			UpdatedAt:   time.Now().UTC().Truncate(time.Microsecond),
		}
		createdItem, err := repo.CreateOrderItem(ctx, item)
		require.NoError(t, err)
		assert.Equal(t, item.Id, createdItem.Id)
		assert.Equal(t, price.Id, createdItem.Price.Id) // Price preloaded
		assert.Equal(t, int64(1999), createdItem.Price.UnitPrice)

		gotItem, err := repo.FindOrderItemById(ctx, orgId, item.Id)
		require.NoError(t, err)
		assert.Equal(t, 2, gotItem.Quantity)
		assert.Equal(t, price.Id, gotItem.Price.Id)

		// Update item.
		gotItem.Quantity = 5
		updatedItem, err := repo.UpdateOrderItem(ctx, gotItem)
		require.NoError(t, err)
		assert.Equal(t, 5, updatedItem.Quantity)

		// FindOrderItemsByOrderId returns the item, and the parent order now
		// preloads its items.
		items, err := repo.FindOrderItemsByOrderId(ctx, orgId, order.Id)
		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, item.Id, items[0].Id)

		gotOrder, err := repo.FindById(ctx, orgId, order.Id)
		require.NoError(t, err)
		require.Len(t, gotOrder.Items, 1)
		assert.Equal(t, item.Id, gotOrder.Items[0].Id)
	})

	t.Run("Find paginates and counts within org", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		cust := seedCustomer(t, db, orgId)
		for range 3 {
			_, err := repo.Create(ctx, seedOrderModel(orgId, cust.Id))
			require.NoError(t, err)
		}
		p := domain.Pagination{Limit: 2, SortBy: "created_at", SortDirection: "asc"}
		orders, count, err := repo.Find(ctx, orgId, p)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
		assert.Len(t, orders, 2)
	})

	t.Run("org-scoping isolates orders", func(t *testing.T) {
		orgA := uniqueOrg(t)
		orgB := uniqueOrg(t)
		cleanupOrg(t, db, orgA)
		cleanupOrg(t, db, orgB)
		cust := seedCustomer(t, db, orgA)
		created, err := repo.Create(ctx, seedOrderModel(orgA, cust.Id))
		require.NoError(t, err)

		_, err = repo.FindById(ctx, orgB, created.Id)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
}

func seedOrderModel(orgId, customerId string) domain.Order {
	return domain.Order{
		OrgId:      orgId,
		Id:         lib.GenerateId("ord"),
		CustomerId: customerId,
		Reference:  "REF-" + lib.GenerateId("r"),
		Status:     domain.OrderStatusPending,
		Currency:   "USD",
		Total:      1999,
		CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}
}
