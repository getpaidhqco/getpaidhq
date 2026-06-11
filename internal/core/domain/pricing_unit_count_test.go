package domain

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// Pins the fixed-scheme unit_count math: UnitPrice cents buy UnitCount units, so
// an integer-cent price expresses a sub-cent effective rate. The quantity is
// multiplied before the division and the total is rounded once, so the rate
// accumulates exactly across the quantity.

func TestPriceUsage_UnitCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		price     Price
		units     string
		wantCents int64
	}{
		{
			name:      "$1 per 1000 calls: 2500 calls bill 250c",
			price:     Price{Scheme: Fixed, UnitPrice: 100, UnitCount: 1000},
			units:     "2500",
			wantCents: 250,
		},
		{
			name:      "sub-cent residue rounds half away from zero",
			price:     Price{Scheme: Fixed, UnitPrice: 100, UnitCount: 1000},
			units:     "2505", // 250.5c
			wantCents: 251,
		},
		{
			name:      "tiny usage on a sub-cent rate rounds to zero",
			price:     Price{Scheme: Fixed, UnitPrice: 1, UnitCount: 1000},
			units:     "5", // 0.005c
			wantCents: 0,
		},
		{
			name:      "large usage on a sub-cent rate accumulates exactly",
			price:     Price{Scheme: Fixed, UnitPrice: 1, UnitCount: 1000},
			units:     "1234567", // 1234.567c
			wantCents: 1235,
		},
		{
			name:      "fractional quantity divides before the single round",
			price:     Price{Scheme: Fixed, UnitPrice: 100, UnitCount: 1000},
			units:     "2.5", // 0.25c
			wantCents: 0,
		},
		{
			name:      "non-terminating rate stays exact via mul-before-div",
			price:     Price{Scheme: Fixed, UnitPrice: 1, UnitCount: 3},
			units:     "3", // 3 × 1/3 = exactly 1c
			wantCents: 1,
		},
		{
			name:      "unit count 1 is plain per-unit pricing",
			price:     Price{Scheme: Fixed, UnitPrice: 999, UnitCount: 1},
			units:     "3.17",
			wantCents: 3167,
		},
		{
			name:      "unit count 0 (legacy rows / zero value) is per-unit pricing",
			price:     Price{Scheme: Fixed, UnitPrice: 999},
			units:     "3.17",
			wantCents: 3167,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := PriceUsage(tt.price, decimal.RequireFromString(tt.units))
			assert.Equal(t, tt.wantCents, got)
		})
	}
}

func TestPriceUsage_UnitCountEffectiveRate(t *testing.T) {
	t.Parallel()
	_, unit := PriceUsage(Price{Scheme: Fixed, UnitPrice: 100, UnitCount: 1000}, decimal.NewFromInt(2500))
	assert.True(t, unit.Equal(decimal.RequireFromString("0.1")), "got %s", unit)
}

func TestFixedLineAmount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                           string
		unitPrice, unitCount, quantity int64
		want                           int64
	}{
		{"unit count 1 multiplies plainly", 999, 1, 3, 2997},
		{"unit count 0 multiplies plainly", 999, 0, 3, 2997},
		{"$1 per 1000: 2500 units", 100, 1000, 2500, 250},
		{"rounds down below the half cent", 1, 1000, 1499, 1},  // 1.499c
		{"rounds up from the half cent", 1, 1000, 1500, 2},     // 1.5c
		{"rounds to zero below half a cent", 1, 1000, 499, 0},  // 0.499c
		{"zero quantity bills zero", 100, 1000, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FixedLineAmount(tt.unitPrice, tt.unitCount, tt.quantity))
		})
	}
}

// A cart line snapshots UnitCount and divides in Calculate; persisted carts from
// before the column unmarshal UnitCount as 0 and keep plain multiplication.
func TestCartLineItem_CalculateUnitCount(t *testing.T) {
	t.Parallel()
	t.Run("divides by the snapshotted unit count", func(t *testing.T) {
		i := CartLineItem{UnitPrice: 100, UnitCount: 1000, Quantity: 2500}
		i.Calculate()
		assert.Equal(t, int64(250), i.SubTotal)
		assert.Equal(t, int64(250), i.Total)
	})
	t.Run("legacy line without unit count multiplies plainly", func(t *testing.T) {
		i := CartLineItem{UnitPrice: 100, Quantity: 3}
		i.Calculate()
		assert.Equal(t, int64(300), i.SubTotal)
	})
}

// The subscription base line divides by UnitCount and carries the fractional
// effective rate as its UnitAmount, so quantity × unit_amount reproduces the total.
func TestBaseLineFromPrice_UnitCount(t *testing.T) {
	t.Parallel()
	p := Price{Id: "price_1", Scheme: Fixed, UnitPrice: 100, UnitCount: 1000, Label: "calls"}
	line := BaseLineFromPrice("org_1", "inv_1", p, decimal.NewFromInt(2500))
	assert.Equal(t, int64(250), line.Total)
	assert.True(t, line.UnitAmount.Equal(decimal.RequireFromString("0.1")), "got %s", line.UnitAmount)
}
