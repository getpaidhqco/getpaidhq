package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cartWith(items ...CartLineItem) *Cart {
	return &Cart{Data: CartData{Items: items}}
}

func TestCart_Calculate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		items        []CartLineItem
		wantSubTotal int64
		wantTotal    int64
		wantTax      int64
		wantDiscount int64
	}{
		{
			name:  "empty cart is zero",
			items: nil,
		},
		{
			name:         "single line item",
			items:        []CartLineItem{{Id: "a", UnitPrice: 1000, Quantity: 3}},
			wantSubTotal: 3000,
			wantTotal:    3000,
		},
		{
			name: "multiple items aggregate",
			items: []CartLineItem{
				{Id: "a", UnitPrice: 1000, Quantity: 2, TaxTotal: 100, DiscountTotal: 50},
				{Id: "b", UnitPrice: 500, Quantity: 1, TaxTotal: 25, DiscountTotal: 10},
			},
			wantSubTotal: 2500,
			wantTotal:    2500, // NOTE: Total currently mirrors SubTotal; tax/discount are
			wantTax:      125,  // tracked separately but NOT folded into Total. Locked as-is.
			wantDiscount: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := cartWith(tt.items...)
			c.Calculate()

			assert.Equal(t, tt.wantSubTotal, c.Data.SubTotal, "sub_total")
			assert.Equal(t, tt.wantTotal, c.Data.Total, "total")
			assert.Equal(t, tt.wantTotal, c.Total, "mirrored Total field")
			assert.Equal(t, tt.wantTax, c.Data.TaxTotal, "tax_total")
			assert.Equal(t, tt.wantDiscount, c.Data.DiscountTotal, "discount_total")

			// Each line item's own SubTotal/Total are recomputed too.
			for i := range c.Data.Items {
				it := c.Data.Items[i]
				assert.Equal(t, it.UnitPrice*it.Quantity, it.SubTotal, "item %s sub_total", it.Id)
				assert.Equal(t, it.SubTotal, it.Total, "item %s total", it.Id)
			}
		})
	}
}

func TestCart_RemoveItem(t *testing.T) {
	t.Parallel()

	t.Run("removes the matching item and recalculates", func(t *testing.T) {
		t.Parallel()
		c := cartWith(
			CartLineItem{Id: "a", UnitPrice: 1000, Quantity: 1},
			CartLineItem{Id: "b", UnitPrice: 500, Quantity: 1},
		)
		c.RemoveItem("a")

		require.Len(t, c.Data.Items, 1)
		assert.Equal(t, "b", c.Data.Items[0].Id)
		assert.Equal(t, int64(500), c.Data.Total)
	})

	t.Run("unknown id is a no-op but still recalculates", func(t *testing.T) {
		t.Parallel()
		c := cartWith(CartLineItem{Id: "a", UnitPrice: 1000, Quantity: 2})
		c.RemoveItem("missing")

		require.Len(t, c.Data.Items, 1)
		assert.Equal(t, int64(2000), c.Data.Total)
	})
}

func TestCart_AdjustQuantity(t *testing.T) {
	t.Parallel()

	t.Run("adjusts quantity and recalculates", func(t *testing.T) {
		t.Parallel()
		c := cartWith(CartLineItem{Id: "a", UnitPrice: 1000, Quantity: 1})

		err := c.AdjustQuantity("a", 4)

		require.NoError(t, err)
		assert.Equal(t, int64(4), c.Data.Items[0].Quantity)
		assert.Equal(t, int64(4000), c.Data.Total)
	})

	t.Run("missing item returns ErrItemNotFound", func(t *testing.T) {
		t.Parallel()
		c := cartWith(CartLineItem{Id: "a", UnitPrice: 1000, Quantity: 1})

		err := c.AdjustQuantity("missing", 4)

		assert.ErrorIs(t, err, ErrItemNotFound)
		assert.Equal(t, int64(1), c.Data.Items[0].Quantity, "quantity unchanged on failure")
	})
}

// TestPriceToCartItemPrice locks the field mapping, including the deliberate
// omissions: MinPrice and SuggestedPrice exist on CartItemPrice but are NOT
// copied from Price, so they must stay zero.
func TestPriceToCartItemPrice(t *testing.T) {
	t.Parallel()

	p := Price{
		Id:                 "price_1",
		Category:           PriceCategory("recurring"),
		Scheme:             PriceScheme("flat"),
		Currency:           Currency("USD"),
		Cycles:             12,
		UnitPrice:          1500,
		MinPrice:           100,
		SuggestedPrice:     2000,
		BillingInterval:    BillingInterval("month"),
		BillingIntervalQty: 1,
		TrialInterval:      BillingInterval("day"),
		TrialIntervalQty:   14,
		TaxCode:            "txcd_123",
	}

	got := PriceToCartItemPrice(p)

	assert.Equal(t, p.Id, got.Id)
	assert.Equal(t, p.Category, got.Category)
	assert.Equal(t, p.Scheme, got.Scheme)
	assert.Equal(t, string(p.Currency), got.Currency)
	assert.Equal(t, int64(p.Cycles), got.Cycles)
	assert.Equal(t, p.UnitPrice, got.UnitPrice)
	assert.Equal(t, p.BillingInterval, got.BillingInterval)
	assert.Equal(t, int64(p.BillingIntervalQty), got.BillingIntervalQty)
	assert.Equal(t, p.TrialInterval, got.TrialInterval)
	assert.Equal(t, int64(p.TrialIntervalQty), got.TrialIntervalQty)
	assert.Equal(t, p.TaxCode, got.TaxCode)

	assert.Zero(t, got.MinPrice, "MinPrice is intentionally not mapped")
	assert.Zero(t, got.SuggestedPrice, "SuggestedPrice is intentionally not mapped")
}
