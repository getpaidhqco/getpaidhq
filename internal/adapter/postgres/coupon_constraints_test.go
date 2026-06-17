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

func TestCouponConstraints(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	orgId := uniqueOrg(t)
	cleanupOrg(t, db, orgId)

	t.Run("rejects both amount_off and percent_off", func(t *testing.T) {
		// Bypass the domain constructor: write a raw invalid row.
		err := db.Exec(`INSERT INTO coupons (org_id, id, name, active, discount_type, amount_off, currency, percent_off, duration, max_redemptions, once_per_customer, created_at, updated_at)
			VALUES (?, ?, 'bad', true, 'fixed', 500, 'USD', 10, 'once', 0, false, now(), now())`,
			orgId, "coupon_bad").Error
		require.Error(t, err, "discount_type XOR check must reject")
	})

	t.Run("immutability trigger blocks term change", func(t *testing.T) {
		c, _ := domain.NewCoupon(domain.NewCouponInput{
			OrgId: orgId, Name: "Imm", DiscountType: domain.DiscountTypePercentage,
			PercentOff: decimal.NewFromInt(10), Duration: domain.DurationForever,
		})
		_, err := NewCouponRepo(db).Create(ctx, c)
		require.NoError(t, err)

		err = db.Exec(`UPDATE coupons SET percent_off = 20 WHERE org_id = ? AND id = ?`, orgId, c.Id).Error
		require.Error(t, err, "trigger must block term update")

		// name change is allowed
		err = db.Exec(`UPDATE coupons SET name = 'Renamed' WHERE org_id = ? AND id = ?`, orgId, c.Id).Error
		assert.NoError(t, err)
	})
}
