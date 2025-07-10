package dto

import (
	"payloop/internal/domain/entities"
	"time"
)

// RecordUsageInput represents input for recording usage
type RecordUsageInput struct {
	OrgId       string                 `json:"org_id"`      // Organization ID
	SpecVersion string                 `json:"specversion"` // Always "1.0"
	Type        string                 `json:"type"`        // Meter type/id
	Id          string                 `json:"id"`          // Unique event identifier
	Time        time.Time              `json:"time"`        // Event timestamp (RFC3339)
	Source      string                 `json:"source"`      // Service/app that generated the event
	Subject     string                 `json:"subject"`     // subscriptionItemId (entity being metered)
	Data        map[string]interface{} `json:"data"`        // Flexible event payload
}

// BatchRecordUsageInput represents input for batch recording usage
type BatchRecordUsageInput struct {
	Records []RecordUsageInput `json:"records"`
}

// UsageSummaryInput represents input for getting usage summary
type UsageSummaryInput struct {
	SubscriptionItemId string    `json:"subscription_item_id"`
	StartDate          time.Time `json:"start_date"`
	EndDate            time.Time `json:"end_date"`
}

// UsageSummaryResult represents usage summary data
type UsageSummaryResult struct {
	SubscriptionId     string                   `json:"subscription_id"`
	SubscriptionItemId string                   `json:"subscription_item_id"`
	BillingPeriod      string                   `json:"billing_period"`
	UsageType          entities.UsageType       `json:"usage_type"`
	UnitType           entities.UnitType        `json:"unit_type"`
	AggregationType    entities.AggregationType `json:"aggregation_type"`
	TotalQuantity      float64                  `json:"total_quantity"`
	TotalAmount        int64                    `json:"total_amount"`
	Details            map[string]interface{}   `json:"details"`
}

// ListUsageRecordsInput represents input for listing usage records
type ListUsageRecordsInput struct {
	SubscriptionItemId string     `json:"subscription_item_id"`
	Pagination         Pagination `json:"pagination"`
}

// GetSubscriptionUsageInput represents input for getting subscription usage
type GetSubscriptionUsageInput struct {
	SubscriptionId string    `json:"subscription_id"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
}

// UsageRecordingResponse represents the immediate response after recording usage
type UsageRecordingResponse struct {
	EventId            string    `json:"event_id"`          // Internal event ID
	OriginalEventId    string    `json:"original_event_id"` // Original CloudEvent ID
	SubscriptionItemId string    `json:"subscription_item_id"`
	Type               string    `json:"type"`   // CloudEvent type
	Status             string    `json:"status"` // "recorded", "processing", "calculated"
	RecordedAt         time.Time `json:"recorded_at"`
}

// GetUsageEstimateInput represents input for getting usage estimate for a subscription
type GetUsageEstimateInput struct {
	SubscriptionId string `json:"subscription_id"`
}

// UsageEstimateResult represents the usage estimate for a subscription
type UsageEstimateResult struct {
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
