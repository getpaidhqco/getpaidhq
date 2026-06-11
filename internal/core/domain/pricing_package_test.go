package domain

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// Pins the package-scheme math: every STARTED block of UnitCount units owes the
// full UnitPrice — ceil(units/UnitCount) × UnitPrice. The round-up sibling of the
// fixed scheme's prorating division; same (UnitPrice, UnitCount) pair.

func TestPriceUsage_Package(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		price     Price
		units     string
		wantCents int64
	}{
		{
			name:      "$5 per started 1000: 12400 units bill 13 blocks",
			price:     Price{Scheme: Package, UnitPrice: 500, UnitCount: 1000},
			units:     "12400",
			wantCents: 6500,
		},
		{
			name:      "exact multiple bills exact blocks",
			price:     Price{Scheme: Package, UnitPrice: 500, UnitCount: 1000},
			units:     "12000",
			wantCents: 6000,
		},
		{
			name:      "one unit into a new block owes the full block",
			price:     Price{Scheme: Package, UnitPrice: 500, UnitCount: 1000},
			units:     "12001",
			wantCents: 6500,
		},
		{
			name:      "tiny usage owes one full block, not a prorated sliver",
			price:     Price{Scheme: Package, UnitPrice: 500, UnitCount: 1000},
			units:     "37",
			wantCents: 500,
		},
		{
			name:      "zero usage owes nothing (no minimum block)",
			price:     Price{Scheme: Package, UnitPrice: 500, UnitCount: 1000},
			units:     "0",
			wantCents: 0,
		},
		{
			name:      "negative usage owes nothing",
			price:     Price{Scheme: Package, UnitPrice: 500, UnitCount: 1000},
			units:     "-5",
			wantCents: 0,
		},
		{
			name:      "fractional quantity still starts the block",
			price:     Price{Scheme: Package, UnitPrice: 500, UnitCount: 1000},
			units:     "1000.5", // weighted_sum meters can yield fractions
			wantCents: 1000,
		},
		{
			name:      "unit count 1 ceils each started unit",
			price:     Price{Scheme: Package, UnitPrice: 500, UnitCount: 1},
			units:     "2.5",
			wantCents: 1500,
		},
		{
			name:      "unit count 0 (zero value) behaves as per-unit blocks",
			price:     Price{Scheme: Package, UnitPrice: 500},
			units:     "2.5",
			wantCents: 1500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := PriceUsage(tt.price, decimal.RequireFromString(tt.units))
			assert.Equal(t, tt.wantCents, got)
		})
	}
}

// The same quantity under fixed prorates the partial block; under package it owes
// the full block — the merchant-visible difference between the two schemes.
func TestPriceUsage_PackageVsFixedPartialBlock(t *testing.T) {
	t.Parallel()
	units := decimal.NewFromInt(12400)
	fixed, _ := PriceUsage(Price{Scheme: Fixed, UnitPrice: 500, UnitCount: 1000}, units)
	pkg, _ := PriceUsage(Price{Scheme: Package, UnitPrice: 500, UnitCount: 1000}, units)
	assert.Equal(t, int64(6200), fixed, "fixed prorates the partial block")
	assert.Equal(t, int64(6500), pkg, "package bills the started block in full")
}

// The usage line's UnitAmount is the effective blended rate (total/units), same
// convention as graduated, so quantity × unit_amount reproduces the line total
// to the cent.
func TestPriceUsage_PackageEffectiveRate(t *testing.T) {
	t.Parallel()
	total, unit := PriceUsage(Price{Scheme: Package, UnitPrice: 500, UnitCount: 1000}, decimal.NewFromInt(12400))
	assert.Equal(t, int64(6500), total)
	roundTrip := unit.Mul(decimal.NewFromInt(12400)).Round(0)
	assert.True(t, roundTrip.Equal(decimal.NewFromInt(6500)), "rate %s × qty rounds to %s, want 6500", unit, roundTrip)
}

func TestUsageLineFromPrice_Package(t *testing.T) {
	t.Parallel()
	p := Price{Id: "price_1", Scheme: Package, UnitPrice: 500, UnitCount: 1000, Label: "sms", BillableMetricId: "met_1"}
	line := UsageLineFromPrice("org_1", "inv_1", p, decimal.NewFromInt(12400))
	assert.Equal(t, int64(6500), line.Total)
	assert.Equal(t, InvoiceLineKindUsage, line.Kind)
	assert.True(t, line.Quantity.Equal(decimal.NewFromInt(12400)))
}
