package domain

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validPercentInput() NewCouponInput {
	return NewCouponInput{
		OrgId:        "org_1",
		Name:         "Summer",
		DiscountType: DiscountTypePercentage,
		PercentOff:   decimal.NewFromInt(25),
		Duration:     DurationForever,
	}
}

func validFixedInput() NewCouponInput {
	return NewCouponInput{
		OrgId:        "org_1",
		Name:         "Tenner",
		DiscountType: DiscountTypeFixed,
		AmountOff:    1000,
		Currency:     "USD",
		Duration:     DurationOnce,
	}
}

func TestNewCoupon(t *testing.T) {
	t.Run("percentage coupon is valid and defaults active", func(t *testing.T) {
		c, err := NewCoupon(validPercentInput())
		require.NoError(t, err)
		assert.True(t, c.Active)
		assert.Contains(t, c.Id, "coup_")
		assert.Equal(t, DiscountTypePercentage, c.DiscountType)
	})

	t.Run("fixed coupon is valid", func(t *testing.T) {
		c, err := NewCoupon(validFixedInput())
		require.NoError(t, err)
		assert.EqualValues(t, 1000, c.AmountOff)
		assert.Equal(t, "USD", c.Currency)
	})

	t.Run("percentage with amount set is rejected", func(t *testing.T) {
		in := validPercentInput()
		in.AmountOff = 500
		_, err := NewCoupon(in)
		require.Error(t, err)
	})

	t.Run("percent out of range is rejected", func(t *testing.T) {
		in := validPercentInput()
		in.PercentOff = decimal.NewFromInt(150)
		_, err := NewCoupon(in)
		require.Error(t, err)
	})

	t.Run("fixed without currency is rejected", func(t *testing.T) {
		in := validFixedInput()
		in.Currency = ""
		_, err := NewCoupon(in)
		require.Error(t, err)
	})

	t.Run("repeating requires cycles >= 1", func(t *testing.T) {
		in := validPercentInput()
		in.Duration = DurationRepeating
		_, err := NewCoupon(in)
		require.Error(t, err)
		in.DurationInCycles = 3
		_, err = NewCoupon(in)
		require.NoError(t, err)
	})

	t.Run("non-repeating must not set cycles", func(t *testing.T) {
		in := validPercentInput()
		in.DurationInCycles = 2
		_, err := NewCoupon(in)
		require.Error(t, err)
	})
}

func TestCouponAppliesTo(t *testing.T) {
	assert.True(t, Coupon{}.appliesTo("prd_x"), "empty scope = whole bill")
	c := Coupon{AppliesToProducts: []string{"prd_a", "prd_b"}}
	assert.True(t, c.appliesTo("prd_b"))
	assert.False(t, c.appliesTo("prd_z"))
}

func TestCouponMutators(t *testing.T) {
	c, _ := NewCoupon(validPercentInput())
	c.Rename("New Name")
	c.SetActive(false)
	c.SetMetadata(map[string]string{"k": "v"})
	assert.Equal(t, "New Name", c.Name)
	assert.False(t, c.Active)
	assert.Equal(t, "v", c.Metadata["k"])
}
