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
	default: // Fixed
		unit := decimal.NewFromInt(p.UnitPrice)
		return unit.Mul(units).Round(0).IntPart(), unit
	}
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
