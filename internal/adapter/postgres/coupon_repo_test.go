//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
)

func TestCouponRepo(t *testing.T) {
	db := testDB(t)
	repo := NewCouponRepo(db)
	ctx := context.Background()

	t.Run("create then find round-trips", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		in, err := domain.NewCoupon(domain.NewCouponInput{
			OrgId: orgId, Name: "Quarter", DiscountType: domain.DiscountTypePercentage,
			PercentOff: decimal.NewFromInt(25), Duration: domain.DurationForever,
			AppliesToProducts: []string{"prd_a", "prd_b"}, Metadata: map[string]string{"k": "v"},
		})
		require.NoError(t, err)

		created, err := repo.Create(ctx, in)
		require.NoError(t, err)
		assert.True(t, decimal.NewFromInt(25).Equal(created.PercentOff))
		assert.Equal(t, []string{"prd_a", "prd_b"}, created.AppliesToProducts)
		assert.Equal(t, map[string]string{"k": "v"}, created.Metadata)

		got, err := repo.FindById(ctx, orgId, in.Id)
		require.NoError(t, err)
		assert.Equal(t, in.Id, got.Id)
	})

	t.Run("update mutable changes only name/active/metadata", func(t *testing.T) {
		orgId := uniqueOrg(t)
		cleanupOrg(t, db, orgId)
		in, _ := domain.NewCoupon(domain.NewCouponInput{
			OrgId: orgId, Name: "Old", DiscountType: domain.DiscountTypeFixed,
			AmountOff: 1000, Currency: "USD", Duration: domain.DurationOnce,
		})
		_, err := repo.Create(ctx, in)
		require.NoError(t, err)

		updated, err := repo.UpdateMutable(ctx, orgId, in.Id, "New", false, map[string]string{"x": "y"})
		require.NoError(t, err)
		assert.Equal(t, "New", updated.Name)
		assert.False(t, updated.Active)
		assert.EqualValues(t, 1000, updated.AmountOff, "terms untouched")
	})
}
