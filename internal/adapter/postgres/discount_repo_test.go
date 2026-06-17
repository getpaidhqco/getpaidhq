//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
)

func TestDiscountRepo(t *testing.T) {
	db := testDB(t)
	repo := NewDiscountRepo(db)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	coupon := seedCoupon(t, db, orgId)

	d, err := domain.NewDiscount(domain.NewDiscountInput{
		OrgId: orgId, CouponId: coupon.Id, CustomerId: "cus_1", SubscriptionId: "sub_1",
	})
	require.NoError(t, err)
	_, err = repo.Create(ctx, d)
	require.NoError(t, err)

	t.Run("active for subscription", func(t *testing.T) {
		got, err := repo.ActiveForSubscription(ctx, orgId, "sub_1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, d.Id, got[0].Id)
	})

	t.Run("counts", func(t *testing.T) {
		byCoupon, err := repo.CountByCoupon(ctx, orgId, coupon.Id)
		require.NoError(t, err)
		assert.Equal(t, 1, byCoupon)
		byCust, err := repo.CountByCouponAndCustomer(ctx, orgId, coupon.Id, "cus_1")
		require.NoError(t, err)
		assert.Equal(t, 1, byCust)
	})
}
