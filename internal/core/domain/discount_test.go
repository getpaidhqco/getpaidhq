package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscount(t *testing.T) {
	t.Run("subscription target is valid", func(t *testing.T) {
		d, err := NewDiscount(NewDiscountInput{OrgId: "org_1", CouponId: "coupon_1", CustomerId: "cus_1", SubscriptionId: "sub_1"})
		require.NoError(t, err)
		assert.Equal(t, DiscountStatusActive, d.Status)
		assert.Contains(t, d.Id, "disc_")
	})

	t.Run("order target is valid", func(t *testing.T) {
		_, err := NewDiscount(NewDiscountInput{OrgId: "org_1", CouponId: "coupon_1", CustomerId: "cus_1", OrderId: "ord_1"})
		require.NoError(t, err)
	})

	t.Run("both targets is rejected", func(t *testing.T) {
		_, err := NewDiscount(NewDiscountInput{OrgId: "org_1", CouponId: "coupon_1", CustomerId: "cus_1", SubscriptionId: "sub_1", OrderId: "ord_1"})
		require.Error(t, err)
	})

	t.Run("no target is rejected", func(t *testing.T) {
		_, err := NewDiscount(NewDiscountInput{OrgId: "org_1", CouponId: "coupon_1", CustomerId: "cus_1"})
		require.Error(t, err)
	})
}
