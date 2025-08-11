package response

import "time"

// CustomerMrrResponse represents the monthly recurring revenue data for a customer
type CustomerMrrResponse struct {
	CustomerId               string                `json:"customer_id"`
	TotalMrr                 int64                 `json:"total_mrr"`
	Currency                 string                `json:"currency"`
	Breakdown                []MrrBreakdownItem    `json:"breakdown"`
	ProjectedAnnualRevenue   int64                 `json:"projected_annual_revenue"`
}

// MrrBreakdownItem represents individual subscription contribution to MRR
type MrrBreakdownItem struct {
	SubscriptionId     string    `json:"subscription_id"`
	ProductName        string    `json:"product_name"`
	MonthlyAmount      int64     `json:"monthly_amount"`
	BillingInterval    string    `json:"billing_interval"`
	NormalizedMonthly  int64     `json:"normalized_monthly,omitempty"`
	NextBilling        time.Time `json:"next_billing"`
}