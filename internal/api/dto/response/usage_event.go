package response

import (
	"payloop/internal/domain/entities"
	"time"
)

// UsageEventResponse represents the API response for a usage event
type UsageEventResponse struct {
	OrgId              string            `json:"org_id"`
	Id                 string            `json:"id"`
	SubscriptionId     string            `json:"subscription_id"`
	SubscriptionItemId string            `json:"subscription_item_id"`
	MeterId            string            `json:"meter_id,omitempty"`

	// Usage details
	UsageType          entities.UsageType `json:"usage_type"`
	Quantity           float64            `json:"quantity"`
	TransactionValue   int64              `json:"transaction_value,omitempty"`
	PercentageRate     float64            `json:"percentage_rate,omitempty"`
	TotalAmount        int64              `json:"total_amount,omitempty"`

	// Timing
	UsageDate          time.Time          `json:"usage_date"`

	// Processing status
	Processed          bool               `json:"processed"`

	// References
	ReferenceId        string            `json:"reference_id,omitempty"`
	ReferenceType      string            `json:"reference_type,omitempty"`

	// Metadata
	Metadata           map[string]string `json:"metadata,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// UsageEventListResponse represents a paginated list of usage events
type UsageEventListResponse struct {
	Items      []UsageEventResponse `json:"items"`
	TotalCount int                  `json:"total_count"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
}

// UsageSummaryResponse represents a summary of usage for a subscription item
type UsageSummaryResponse struct {
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
