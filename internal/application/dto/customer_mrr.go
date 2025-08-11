package dto

import "time"

// CustomerMrrData represents the calculated MRR data for internal use
type CustomerMrrData struct {
	CustomerId               string
	TotalMrr                 int64
	Currency                 string
	Breakdown                []MrrBreakdownData
	ProjectedAnnualRevenue   int64
}

// MrrBreakdownData represents individual subscription MRR contribution
type MrrBreakdownData struct {
	SubscriptionId     string
	ProductName        string
	MonthlyAmount      int64
	BillingInterval    string
	NormalizedMonthly  int64
	NextBilling        time.Time
}