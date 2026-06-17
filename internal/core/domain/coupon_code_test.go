package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCouponCode(t *testing.T) {
	t.Run("valid code uppercases and defaults active", func(t *testing.T) {
		cc, err := NewCouponCode(NewCouponCodeInput{OrgId: "org_1", CouponId: "coupon_1", Code: "summer25"})
		require.NoError(t, err)
		assert.Equal(t, "SUMMER25", cc.Code)
		assert.True(t, cc.Active)
		assert.Contains(t, cc.Id, "ccode_")
	})

	t.Run("missing code is rejected", func(t *testing.T) {
		_, err := NewCouponCode(NewCouponCodeInput{OrgId: "org_1", CouponId: "coupon_1"})
		require.Error(t, err)
	})

	t.Run("negative cap is rejected", func(t *testing.T) {
		_, err := NewCouponCode(NewCouponCodeInput{OrgId: "org_1", CouponId: "coupon_1", Code: "X", MaxRedemptions: -1})
		require.Error(t, err)
	})

	t.Run("minimum amount requires currency", func(t *testing.T) {
		_, err := NewCouponCode(NewCouponCodeInput{
			OrgId: "org_1", CouponId: "coupon_1", Code: "X",
			Restrictions: Restrictions{MinimumAmount: 500},
		})
		require.Error(t, err)
	})
}
