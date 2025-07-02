package dto

import (
	"payloop/internal/domain/entities"
	"time"
)

// RecordUsageInput represents input for recording usage
type RecordUsageInput struct {
	SubscriptionItemId string            `json:"subscription_item_id"`
	Quantity           float64           `json:"quantity"`
	TransactionValue   *int64            `json:"transaction_value,omitempty"`
	PercentageRate     *float64          `json:"percentage_rate,omitempty"`
	ReferenceId        string            `json:"reference_id,omitempty"`
	ReferenceType      string            `json:"reference_type,omitempty"`
	Timestamp          time.Time         `json:"timestamp"`
	Metadata           map[string]string `json:"metadata,omitempty"`
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
