package response

// UsageEstimateResponse represents the usage estimate for a subscription
type UsageEstimateResponse struct {
	SubscriptionId string                 `json:"subscription_id"`
	BaseAmount     int64                  `json:"base_amount"`
	UsageAmount    int64                  `json:"usage_amount"`
	TotalAmount    int64                  `json:"total_amount"`
	Currency       string                 `json:"currency"`
	UsageBreakdown []UsageBreakdownItem   `json:"usage_breakdown"`
	ItemBreakdown  []SubscriptionItemCost `json:"item_breakdown"`
}

// UsageBreakdownItem represents a single item in the usage breakdown
type UsageBreakdownItem struct {
	SubscriptionItemId string  `json:"subscription_item_id"`
	UnitType           string  `json:"unit_type"`
	Quantity           float64 `json:"quantity"`
	UnitPrice          int64   `json:"unit_price"`
	Amount             int64   `json:"amount"`
	AggregationType    string  `json:"aggregation_type"`
}

// SubscriptionItemCost represents the cost for a subscription item
type SubscriptionItemCost struct {
	SubscriptionItemId string `json:"subscription_item_id"`
	Description        string `json:"description"`
	PriceCategory      string `json:"price_category"`
	Amount             int64  `json:"amount"`
}