package domain

import (
	"sort"

	"github.com/shopspring/decimal"
)

// DiscountableLine is the minimal line view ApplyDiscounts needs; the caller
// resolves ProductId (Price -> Variant -> Product) before calling.
type DiscountableLine struct {
	LineId    string
	ProductId string
	Total     int64 // gross line total, minor units
}

// AppliedDiscount pairs a Discount with its (immutable) Coupon.
type AppliedDiscount struct {
	Discount Discount
	Coupon   Coupon
}

// InWindow reports whether this discount applies to the given billing cycle.
func (a AppliedDiscount) InWindow(cycle int) bool {
	switch a.Coupon.Duration {
	case DurationForever:
		return cycle >= a.Discount.StartCycle
	case DurationOnce:
		return cycle == a.Discount.StartCycle
	case DurationRepeating:
		return cycle >= a.Discount.StartCycle && cycle < a.Discount.StartCycle+a.Coupon.DurationInCycles
	default:
		return false
	}
}

// ApplyDiscounts returns the discount amount to record per line id. Pure and
// deterministic. Discounts apply in RedeemedAt order against each line's
// running net, so cumulative discount never drives a line below zero.
func ApplyDiscounts(lines []DiscountableLine, applied []AppliedDiscount, cycle int, currency string) map[string]int64 {
	result := make(map[string]int64, len(lines))
	net := make(map[string]int64, len(lines))
	for _, l := range lines {
		net[l.LineId] = l.Total
		result[l.LineId] = 0
	}

	sorted := append([]AppliedDiscount(nil), applied...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Discount.RedeemedAt.Before(sorted[j].Discount.RedeemedAt)
	})

	for _, a := range sorted {
		if !a.InWindow(cycle) {
			continue
		}
		c := a.Coupon
		if c.DiscountType == DiscountTypeFixed && c.Currency != currency {
			continue
		}

		matched := make([]string, 0, len(lines))
		var base int64
		for _, l := range lines {
			if c.appliesTo(l.ProductId) {
				matched = append(matched, l.LineId)
				base += net[l.LineId]
			}
		}
		if base <= 0 {
			continue
		}

		var raw int64
		switch c.DiscountType {
		case DiscountTypePercentage:
			raw = decimal.NewFromInt(base).Mul(c.PercentOff).Div(decimal.NewFromInt(100)).Round(0).IntPart()
		case DiscountTypeFixed:
			raw = c.AmountOff
			if raw > base {
				raw = base
			}
		}
		if raw <= 0 {
			continue
		}

		for id, amt := range allocateProportional(raw, matched, net) {
			net[id] -= amt
			result[id] += amt
		}
	}
	return result
}

// allocateProportional splits total across lines in proportion to their running
// net, using largest-remainder rounding so the parts sum to total exactly.
func allocateProportional(total int64, lineIds []string, net map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(lineIds))
	var sum int64
	for _, id := range lineIds {
		sum += net[id]
	}
	if sum <= 0 {
		return out
	}

	type rem struct {
		id        string
		remainder int64
	}
	rems := make([]rem, 0, len(lineIds))
	var allocated int64
	for _, id := range lineIds {
		num := net[id] * total
		share := num / sum
		out[id] = share
		allocated += share
		rems = append(rems, rem{id: id, remainder: num % sum})
	}

	leftover := total - allocated
	sort.SliceStable(rems, func(i, j int) bool { return rems[i].remainder > rems[j].remainder })
	for i := int64(0); i < leftover && int(i) < len(rems); i++ {
		out[rems[i].id]++
	}
	return out
}
