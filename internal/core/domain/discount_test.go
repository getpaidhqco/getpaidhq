package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscount(t *testing.T) {
	t.Run("order target is valid", func(t *testing.T) {
		d, err := NewDiscount(NewDiscountInput{OrgId: "org_1", CouponId: "coupon_1", CustomerId: "cus_1", OrderId: "ord_1"})
		require.NoError(t, err)
		assert.Equal(t, DiscountStatusActive, d.Status)
		assert.Contains(t, d.Id, "disc_")
		assert.Equal(t, "ord_1", d.OrderId)
		assert.Empty(t, d.SubscriptionId)
	})

	t.Run("order and subscription target is valid", func(t *testing.T) {
		d, err := NewDiscount(NewDiscountInput{OrgId: "org_1", CouponId: "coupon_1", CustomerId: "cus_1", OrderId: "ord_1", SubscriptionId: "sub_1"})
		require.NoError(t, err)
		assert.Equal(t, "ord_1", d.OrderId)
		assert.Equal(t, "sub_1", d.SubscriptionId)
	})

	t.Run("subscription without order is rejected", func(t *testing.T) {
		_, err := NewDiscount(NewDiscountInput{OrgId: "org_1", CouponId: "coupon_1", CustomerId: "cus_1", SubscriptionId: "sub_1"})
		require.Error(t, err)
	})

	t.Run("no order is rejected", func(t *testing.T) {
		_, err := NewDiscount(NewDiscountInput{OrgId: "org_1", CouponId: "coupon_1", CustomerId: "cus_1"})
		require.Error(t, err)
	})
}
