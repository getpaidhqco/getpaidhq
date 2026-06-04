package domain

import "time"

// BillableMetric (the "meter") defines what customer usage to measure and how to add
// it up over a billing period.
type BillableMetric struct {
	OrgId         string
	Id            string
	Code          string // events reference this; unique per org
	Name          string
	Aggregation   AggregationType
	FieldName     string // which event Metadata key to read; empty for count
	Recurring     bool   // does the running total carry across billing periods
	RoundingMode  string // round | ceil | floor | "" (none)
	RoundingScale int    // decimal places for rounding the aggregated quantity
	Metadata      map[string]string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
