package domain

import "time"

// Price is a billing rule attached to a Variant. Holds the unit/min/suggested
// amounts, the billing interval, and the pricing scheme (fixed / graduated /
// volume).
type Price struct {
	OrgId              string
	Id                 string
	VariantId          string
	Label              string
	Category           PriceCategory
	Scheme             PriceScheme
	Cycles             int
	Currency           Currency
	UnitPrice          int64
	// UnitCount is how many units UnitPrice buys (fixed scheme only): the effective
	// rate is UnitPrice/UnitCount cents per unit, so an integer-cent price can express
	// sub-cent rates ("$1 per 1000 calls" = UnitPrice 100, UnitCount 1000). 0 and 1
	// both mean per single unit.
	UnitCount          int
	MinPrice           int64
	SuggestedPrice     int64
	BillingInterval    BillingInterval
	BillingIntervalQty int
	TrialInterval      BillingInterval
	TrialIntervalQty   int
	TaxCode            string
	BillableMetricId   string      // set when the price is metered: the meter usage is measured against
	Tiers              []PriceTier // rate bands for Graduated / Volume schemes
	// FilterField/FilterValue scope a metered price to one slice of its meter (a value
	// of one of the meter's MetricFilters). FilterField == "" bills the whole meter;
	// FilterField set with FilterValue == "" is the default/catch-all charge (NOT IN
	// the field's declared values). See usage-filters-and-groups.md.
	FilterField string
	FilterValue string
	// Proration switches for prices on time-weighted (weighted_sum) carry-over
	// meters; inert otherwise. See docs/internal/billing-model/seat-billing/mapping.md §2.3.
	ProrateOnIncrease bool // a seat added mid-period accrues from its add date
	CreditOnDecrease  bool // a seat removed mid-period stops accruing at its remove date
	Metadata          map[string]string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// IsDefaultFilter reports whether a metered price is the catch-all charge for its
// filter field (the field is set but no specific value is named).
func (p Price) IsDefaultFilter() bool { return p.FilterField != "" && p.FilterValue == "" }

// IsMetered reports whether the price is usage-based — i.e. it has a meter attached.
// Metering is a pricing method, orthogonal to the price Category (cadence): a metered
// price is typically a recurring subscription billed by usage.
func (p Price) IsMetered() bool { return p.BillableMetricId != "" }

// IsRecurring reports whether the price bills on a cadence (and therefore belongs
// to a subscription). A metered price always recurs (usage must be billed); a
// non-metered price recurs only when it carries a billing interval.
func (p Price) IsRecurring() bool {
	return p.IsMetered() || (p.BillingInterval != "" && p.BillingInterval != BillingIntervalNone)
}

// SubscriptionCadence is the cadence a subscription bills this line at. For a
// non-metered line it's the configured interval. **Metered lines are capped at
// monthly** — usage must never accumulate on a longer cadence (billing a year of
// unbilled usage at once is an unacceptable credit risk), so any metered cadence
// of more than a month (or none) is clamped to monthly; shorter cadences are kept.
func (p Price) SubscriptionCadence() (BillingInterval, int) {
	interval, qty := p.BillingInterval, p.BillingIntervalQty
	if p.IsMetered() {
		if interval == "" || interval == BillingIntervalNone || approxBillingDays(interval, qty) > 31 {
			return BillingIntervalMonth, 1
		}
	}
	return interval, qty
}

// approxBillingDays is a rough day-count for a cadence, used only to compare
// cadence lengths (e.g. is a metered line billing less often than monthly).
func approxBillingDays(interval BillingInterval, qty int) float64 {
	if qty <= 0 {
		qty = 1
	}
	q := float64(qty)
	switch interval {
	case BillingIntervalSecond:
		return q / 86400
	case BillingIntervalMinute:
		return q / 1440
	case BillingIntervalDay:
		return q
	case BillingIntervalWeek:
		return q * 7
	case BillingIntervalMonth:
		return q * 30
	case BillingIntervalYear:
		return q * 365
	default:
		return 0
	}
}
