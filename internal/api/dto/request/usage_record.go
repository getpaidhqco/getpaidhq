package request

import (
	"time"
)

// RecordUsageRequest represents a request to record usage for a subscription item
type RecordUsageRequest struct {
	SpecVersion string                 `json:"specversion"`                // Always "1.0"
	Type        string                 `json:"type" binding:"required"`    // Meter type/id
	Id          string                 `json:"id" binding:"required"`      // Unique event identifier
	Time        time.Time              `json:"time" binding:"required"`    // Event timestamp (RFC3339)
	Source      string                 `json:"source"`                     // Service/app that generated the event
	Subject     string                 `json:"subject" binding:"required"` // subscriptionItemId (entity being metered)
	Data        map[string]interface{} `json:"data" binding:"required"`    // Flexible event payload
}

// BatchRecordUsageRequest represents a request to record multiple usage records
type BatchRecordUsageRequest struct {
	OrgId   string               `json:"org_id"`
	Records []RecordUsageRequest `json:"records" binding:"required,min=1"`
}

// GetUsageSummaryRequest represents a request to get a summary of usage for a subscription item
type GetUsageSummaryRequest struct {
	OrgId              string    `json:"org_id"`
	SubscriptionItemId string    `json:"subscription_item_id" binding:"required"`
	StartDate          time.Time `json:"start_date" binding:"required"`
	EndDate            time.Time `json:"end_date" binding:"required"`
}

// GetUsageByPeriodRequest represents a request to get usage for a billing period
type GetUsageByPeriodRequest struct {
	OrgId          string `json:"org_id"`
	SubscriptionId string `json:"subscription_id" binding:"required"`
	BillingPeriod  string `json:"billing_period" binding:"required"` // Format: YYYY-MM
}
