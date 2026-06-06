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
	MinPrice           int64
	SuggestedPrice     int64
	BillingInterval    BillingInterval
	BillingIntervalQty int
	TrialInterval      BillingInterval
	TrialIntervalQty   int
	TaxCode            string
	BillableMetricId   string      // set when the price is metered: the meter usage is measured against
	Tiers              []PriceTier // rate bands for Graduated / Volume schemes
	Metadata           map[string]string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// IsMetered reports whether the price is usage-based — i.e. it has a meter attached.
// Metering is a pricing method, orthogonal to the price Category (cadence): a metered
// price is typically a recurring subscription billed by usage.
func (p Price) IsMetered() bool { return p.BillableMetricId != "" }
