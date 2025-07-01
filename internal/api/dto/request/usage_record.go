package request

import (
	"time"
)

// RecordUsageRequest represents a request to record usage for a subscription item
type RecordUsageRequest struct {
	OrgId              string            `json:"org_id"`
	SubscriptionItemId string            `json:"subscription_item_id" binding:"required"`
	
	// Unit-based usage
	Quantity           float64           `json:"quantity,omitempty"`
	
	// Percentage-based usage
	TransactionValue   int64             `json:"transaction_value,omitempty"`
	PercentageRate     float64           `json:"percentage_rate,omitempty"`
	
	// Time tracking
	Timestamp          time.Time         `json:"timestamp"`
	
	// External references
	ReferenceId        string            `json:"reference_id,omitempty"`
	ReferenceType      string            `json:"reference_type,omitempty"`
	
	// Metadata
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// BatchRecordUsageRequest represents a request to record multiple usage records
type BatchRecordUsageRequest struct {
	OrgId  string             `json:"org_id"`
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
	OrgId         string `json:"org_id"`
	SubscriptionId string `json:"subscription_id" binding:"required"`
	BillingPeriod string `json:"billing_period" binding:"required"` // Format: YYYY-MM
}