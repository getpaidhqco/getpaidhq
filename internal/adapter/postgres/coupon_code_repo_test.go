//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
)

func TestCouponCodeRepo(t *testing.T) {
	db := testDB(t)
	repo := NewCouponCodeRepo(db)
	ctx := context.Background()

	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	coupon := seedCoupon(t, db, orgId)

	cc, err := domain.NewCouponCode(domain.NewCouponCodeInput{
		OrgId: orgId, CouponId: coupon.Id, Code: "summer25",
		Restrictions: domain.Restrictions{FirstTimeTransaction: true},
	})
	require.NoError(t, err)
	_, err = repo.Create(ctx, cc)
	require.NoError(t, err)

	t.Run("find by code is case-insensitive", func(t *testing.T) {
		got, err := repo.FindByCode(ctx, orgId, "SuMmEr25")
		require.NoError(t, err)
		assert.Equal(t, cc.Id, got.Id)
		assert.True(t, got.Restrictions.FirstTimeTransaction)
	})

	t.Run("increment redeemed", func(t *testing.T) {
		require.NoError(t, repo.IncrementRedeemed(ctx, orgId, cc.Id))
		got, _ := repo.FindByCode(ctx, orgId, "SUMMER25")
		assert.Equal(t, 1, got.TimesRedeemed)
	})
}
