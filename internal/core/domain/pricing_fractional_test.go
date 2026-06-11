package domain

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// Pins how fractional quantities (time-weighted seats, summed decimals) price
// through each scheme, especially across tier boundaries, and how the single
// round-to-cents behaves. These are the behaviors carry-over billing relies on.

func twoTiers() []PriceTier {
	// 0–10 units at 10c, beyond at 5c.
	return []PriceTier{
		{FromValue: decimal.Zero, ToValue: decimal.NewFromInt(10), PerUnitAmount: decimal.NewFromInt(10)},
		{FromValue: decimal.NewFromInt(10), ToValue: decimal.Zero, PerUnitAmount: decimal.NewFromInt(5)},
	}
}

func TestPriceUsage_FractionalQuantities(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		price     Price
		units     string
		wantCents int64
	}{
		{
			name:      "fixed: fractional amount rounds half away from zero",
			price:     Price{Scheme: Fixed, UnitPrice: 999},
			units:     "3.17", // 3166.83c
			wantCents: 3167,
		},
		{
			name:      "fixed: half a cent rounds up",
			price:     Price{Scheme: Fixed, UnitPrice: 5},
			units:     "2.5", // 12.5c
			wantCents: 13,
		},
		{
			name:      "graduated: fraction crosses the tier boundary",
			price:     Price{Scheme: Graduated, Tiers: twoTiers()},
			units:     "10.5", // 10×10c + 0.5×5c = 102.5c
			wantCents: 103,
		},
		{
			name:      "graduated: exactly on the boundary stays in the first tier",
			price:     Price{Scheme: Graduated, Tiers: twoTiers()},
			units:     "10",
			wantCents: 100,
		},
		{
			name:      "graduated: rounding happens once on the total, not per tier",
			price:     Price{Scheme: Graduated, Tiers: twoTiers()},
			units:     "10.1", // 100 + 0.5c = 100.5c → one round → 101 (not 100+1)
			wantCents: 101,
		},
		{
			name:      "volume: fractional total lands in the band it reaches",
			price:     Price{Scheme: Volume, Tiers: twoTiers()},
			units:     "10.5", // whole quantity at the 5c band: 52.5c
			wantCents: 53,
		},
		{
			name:      "volume: exactly on the boundary takes the lower band",
			price:     Price{Scheme: Volume, Tiers: twoTiers()},
			units:     "10", // 10 ≤ ToValue of band 1 → 10×10c
			wantCents: 100,
		},
		{
			name:      "zero units bills zero (graduated)",
			price:     Price{Scheme: Graduated, Tiers: twoTiers()},
			units:     "0",
			wantCents: 0,
		},
		{
			name:      "zero units bills zero (fixed)",
			price:     Price{Scheme: Fixed, UnitPrice: 1000},
			units:     "0",
			wantCents: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := PriceUsage(tt.price, decimal.RequireFromString(tt.units))
			assert.Equal(t, tt.wantCents, got)
		})
	}
}

// The effective per-unit rate is derived from the ROUNDED total, so the line's
// quantity × unit_amount reproduces the billed cents.
func TestPriceUsage_EffectiveUnitRate(t *testing.T) {
	t.Parallel()
	t.Run("volume returns the band's rate", func(t *testing.T) {
		_, unit := PriceUsage(Price{Scheme: Volume, Tiers: twoTiers()}, decimal.RequireFromString("10.5"))
		assert.True(t, unit.Equal(decimal.NewFromInt(5)), "got %s", unit)
	})
	t.Run("graduated returns rounded-total ÷ units", func(t *testing.T) {
		amt, unit := PriceUsage(Price{Scheme: Graduated, Tiers: twoTiers()}, decimal.RequireFromString("10.5"))
		back := unit.Mul(decimal.RequireFromString("10.5")).Round(0).IntPart()
		assert.Equal(t, amt, back, "unit rate must reproduce the billed total")
	})
	t.Run("zero units yields a zero unit rate, not a division error", func(t *testing.T) {
		_, unit := PriceUsage(Price{Scheme: Graduated, Tiers: twoTiers()}, decimal.Zero)
		assert.True(t, unit.IsZero(), "got %s", unit)
	})
}

// A used tier's FlatAmount is added once per tier (graduated) or once for the
// reached band (volume).
func TestPriceUsage_TierFlatAmounts(t *testing.T) {
	t.Parallel()
	tiers := []PriceTier{
		{FromValue: decimal.Zero, ToValue: decimal.NewFromInt(10), PerUnitAmount: decimal.NewFromInt(10), FlatAmount: 100},
		{FromValue: decimal.NewFromInt(10), ToValue: decimal.Zero, PerUnitAmount: decimal.NewFromInt(5), FlatAmount: 50},
	}
	t.Run("graduated adds each used tier's flat", func(t *testing.T) {
		got, _ := PriceUsage(Price{Scheme: Graduated, Tiers: tiers}, decimal.RequireFromString("10.5"))
		// 10×10 + 0.5×5 + 100 + 50 = 252.5, rounded ONCE at the end → 253.
		assert.Equal(t, int64(253), got)
	})
	t.Run("volume adds only the reached band's flat", func(t *testing.T) {
		got, _ := PriceUsage(Price{Scheme: Volume, Tiers: tiers}, decimal.RequireFromString("10.5"))
		assert.Equal(t, int64(53+50), got)
	})
}
