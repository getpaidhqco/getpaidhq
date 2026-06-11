package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestPriceUsage_Fixed(t *testing.T) {
	t.Parallel()
	p := Price{Scheme: Fixed, UnitPrice: 10} // 10 cents/unit
	amt, unit := PriceUsage(p, decimal.NewFromInt(5000))
	if amt != 50000 {
		t.Fatalf("fixed total: want 50000, got %d", amt)
	}
	if !unit.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("fixed unit: want 10, got %s", unit)
	}
}

func TestPriceUsage_Graduated(t *testing.T) {
	t.Parallel()
	// first 100 units @ 10c (+ $2 flat=200), next units @ 5c
	p := Price{Scheme: Graduated, Tiers: []PriceTier{
		{FromValue: decimal.NewFromInt(0), ToValue: decimal.NewFromInt(100), PerUnitAmount: decimal.NewFromInt(10), FlatAmount: 200},
		{FromValue: decimal.NewFromInt(100), ToValue: decimal.Zero, PerUnitAmount: decimal.NewFromInt(5)},
	}}
	// 150 units: 100*10 + 200 (tier1) + 50*5 (tier2) = 1000 + 200 + 250 = 1450
	amt, _ := PriceUsage(p, decimal.NewFromInt(150))
	if amt != 1450 {
		t.Fatalf("graduated total: want 1450, got %d", amt)
	}
	// 10 units: only tier1 → 10*10 + 200 = 300
	amt2, _ := PriceUsage(p, decimal.NewFromInt(10))
	if amt2 != 300 {
		t.Fatalf("graduated small total: want 300, got %d", amt2)
	}
}

func TestPriceUsage_Volume(t *testing.T) {
	t.Parallel()
	// all units at the single tier the total reaches
	p := Price{Scheme: Volume, Tiers: []PriceTier{
		{FromValue: decimal.NewFromInt(0), ToValue: decimal.NewFromInt(100), PerUnitAmount: decimal.NewFromInt(2), FlatAmount: 1000},
		{FromValue: decimal.NewFromInt(101), ToValue: decimal.Zero, PerUnitAmount: decimal.NewFromInt(1)},
	}}
	// 300 units → tier2 (101+): 300*1 = 300
	amt, unit := PriceUsage(p, decimal.NewFromInt(300))
	if amt != 300 {
		t.Fatalf("volume total: want 300, got %d", amt)
	}
	if !unit.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("volume unit: want 1, got %s", unit)
	}
	// 50 units → tier1: 50*2 + 1000 flat = 1100
	amt2, _ := PriceUsage(p, decimal.NewFromInt(50))
	if amt2 != 1100 {
		t.Fatalf("volume tier1 total: want 1100, got %d", amt2)
	}
}

func TestPriceUsage_Tiered_AliasesGraduated(t *testing.T) {
	t.Parallel()
	p := Price{Scheme: Tiered, Tiers: []PriceTier{
		{FromValue: decimal.Zero, ToValue: decimal.Zero, PerUnitAmount: decimal.NewFromInt(3)},
	}}
	amt, _ := PriceUsage(p, decimal.NewFromInt(10))
	if amt != 30 {
		t.Fatalf("tiered(=graduated) total: want 30, got %d", amt)
	}
}
