package domain

import "time"

// MetricFilter declares one filterable dimension of a meter: a metadata key and the
// enumerated values that each get their own priced charge. A Price selects one value
// (its rate); the default/catch-all charge bills everything NOT IN these values. See
// docs/internal/usage-filters-and-groups.md.
type MetricFilter struct {
	Field  string   // event Metadata key, e.g. "type"
	Values []string // the values that get a dedicated Price; default charge = NOT IN these
}

// BillableMetric (the "meter") defines what customer usage to measure and how to add
// it up over a billing period.
type BillableMetric struct {
	OrgId         string
	Id            string
	Code          string // events reference this; unique per org
	Name          string
	Aggregation   AggregationType
	FieldName     string // which event Metadata key to read; empty for count
	CarryOver     bool   // carry the aggregate forward across period boundaries (standing value snapshotted per period) instead of resetting each period
	RoundingMode  string // round | ceil | floor | "" (none)
	RoundingScale int    // decimal places for rounding the aggregated quantity
	// Filters are the rate dimensions: each declares a metadata key + the values that
	// get their own Price. Group is an open breakout dimension (key only): usage is
	// split into one invoice line per discovered value, all at the Price's single rate.
	// Filter sets the rate; Group only itemises. (usage-filters-and-groups.md.)
	Filters   []MetricFilter
	GroupBy   []string
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FilterValues returns the declared values for a filter field, or nil if the field is
// not a declared filter. Used to compute the default charge's NOT-IN exclude set so a
// catch-all Price never has to inspect its sibling Prices.
func (m BillableMetric) FilterValues(field string) []string {
	for _, f := range m.Filters {
		if f.Field == field {
			return f.Values
		}
	}
	return nil
}
