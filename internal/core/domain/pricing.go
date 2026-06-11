package domain

import "github.com/shopspring/decimal"

// PriceTier is one rate band for Graduated / Volume schemes. Amounts are cents;
// PerUnitAmount may be sub-cent. ToValue zero means unbounded (the last/open tier).
type PriceTier struct {
	FromValue     decimal.Decimal // inclusive lower bound, in units
	ToValue       decimal.Decimal // upper bound; zero = unbounded
	PerUnitAmount decimal.Decimal // cents per unit
	FlatAmount    int64           // flat cents added when the tier is used
}

// PriceUsage turns a quantity of units into money for a Price, switching on its
// scheme. Returns the total in whole cents (rounded once) and the effective per-unit
// rate in cents (may be fractional). Shared by the metered-usage path and the
// fixed base-line builder.
func PriceUsage(p Price, units decimal.Decimal) (amountCents int64, unitAmountCents decimal.Decimal) {
	switch p.Scheme {
	case Volume:
		return priceVolume(p.Tiers, units)
	case Graduated, Tiered: // Tiered is an alias for Graduated
		return priceGraduated(p.Tiers, units)
	case Package:
		return pricePackage(p, units)
	default: // Fixed
		return priceFixed(p, units)
	}
}

// pricePackage bills every started block of UnitCount units at UnitPrice cents:
// ceil(units / UnitCount) × UnitPrice. A partial block owes the full block —
// the opposite of priceFixed, which prorates it. Zero usage owes nothing (no
// minimum block). UnitCount <= 1 degenerates to per-unit blocks, so fractional
// quantities still round up to whole units.
func pricePackage(p Price, units decimal.Decimal) (int64, decimal.Decimal) {
	if units.LessThanOrEqual(decimal.Zero) {
		return 0, decimal.Zero
	}
	size := int64(p.UnitCount)
	if size < 1 {
		size = 1
	}
	blocks := units.Div(decimal.NewFromInt(size)).Ceil()
	return roundWithUnit(blocks.Mul(decimal.NewFromInt(p.UnitPrice)), units)
}

// priceFixed bills units at UnitPrice cents per UnitCount units (UnitCount <= 1
// means per single unit). The division happens before the single rounding, so a
// sub-cent effective rate accumulates exactly across the quantity.
func priceFixed(p Price, units decimal.Decimal) (int64, decimal.Decimal) {
	unit := decimal.NewFromInt(p.UnitPrice)
	if p.UnitCount > 1 {
		count := decimal.NewFromInt(int64(p.UnitCount))
		return unit.Mul(units).Div(count).Round(0).IntPart(), unit.Div(count)
	}
	return unit.Mul(units).Round(0).IntPart(), unit
}

// FixedLineAmount is the whole-cent charge for an integer quantity at unitPrice
// cents per unitCount units — the cart/order line total. unitCount <= 1 keeps
// plain integer multiplication.
func FixedLineAmount(unitPrice, unitCount, quantity int64) int64 {
	if unitCount <= 1 {
		return unitPrice * quantity
	}
	return decimal.NewFromInt(unitPrice).
		Mul(decimal.NewFromInt(quantity)).
		Div(decimal.NewFromInt(unitCount)).
		Round(0).IntPart()
}

// priceGraduated bills each unit at the rate of the tier it falls into; total is the
// sum across tiers, plus each used tier's flat amount.
func priceGraduated(tiers []PriceTier, units decimal.Decimal) (int64, decimal.Decimal) {
	total := decimal.Zero
	for _, t := range tiers {
		unbounded := t.ToValue.IsZero()
		hi := units
		if !unbounded && units.GreaterThan(t.ToValue) {
			hi = t.ToValue
		}
		qty := hi.Sub(t.FromValue)
		if qty.LessThanOrEqual(decimal.Zero) {
			continue
		}
		total = total.Add(qty.Mul(t.PerUnitAmount)).Add(decimal.NewFromInt(t.FlatAmount))
		if unbounded || units.LessThanOrEqual(t.ToValue) {
			break
		}
	}
	return roundWithUnit(total, units)
}

// priceVolume bills all units at the single tier the total quantity reaches.
func priceVolume(tiers []PriceTier, units decimal.Decimal) (int64, decimal.Decimal) {
	for _, t := range tiers {
		unbounded := t.ToValue.IsZero()
		if units.GreaterThanOrEqual(t.FromValue) && (unbounded || units.LessThanOrEqual(t.ToValue)) {
			amt := units.Mul(t.PerUnitAmount).Round(0).IntPart() + t.FlatAmount
			return amt, t.PerUnitAmount
		}
	}
	return 0, decimal.Zero
}

func roundWithUnit(total, units decimal.Decimal) (int64, decimal.Decimal) {
	amt := total.Round(0).IntPart()
	if units.LessThanOrEqual(decimal.Zero) {
		return amt, decimal.Zero
	}
	return amt, decimal.NewFromInt(amt).Div(units)
}
