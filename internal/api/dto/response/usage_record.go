package response

import (
	"payloop/internal/domain/entities"
	"time"
)

// UsageRecordResponse represents a response containing a usage record
type UsageRecordResponse struct {
	OrgId              string            `json:"org_id"`
	Id                 string            `json:"id"`
	SubscriptionId     string            `json:"subscription_id"`
	SubscriptionItemId string            `json:"subscription_item_id"`
	CustomerId         string            `json:"customer_id"`

	// Link to price configuration
	PriceId            string            `json:"price_id"`

	// Usage identification
	UsageType          entities.UsageType         `json:"usage_type"`
	UnitType           entities.UnitType          `json:"unit_type,omitempty"`
	AggregationType    entities.AggregationType   `json:"aggregation_type,omitempty"`

	// Unit-based usage
	Quantity           float64           `json:"quantity,omitempty"`
	UnitPrice          int64             `json:"unit_price,omitempty"`

	// Percentage-based usage
	TransactionValue   int64             `json:"transaction_value,omitempty"`
	PercentageRate     float64           `json:"percentage_rate,omitempty"`
	CalculatedFee      int64             `json:"calculated_fee,omitempty"`

	// Hybrid pricing
	FixedFee           int64             `json:"fixed_fee,omitempty"`

	// Final billing amount
	TotalAmount        int64             `json:"total_amount"`

	// Time tracking
	UsageDate          time.Time         `json:"usage_date"`
	BillingPeriod      string            `json:"billing_period"`

	// Processing status
	Processed          bool              `json:"processed"`
	ProcessedAt        time.Time         `json:"processed_at,omitempty"`
	InvoiceId          string            `json:"invoice_id,omitempty"`

	// External references
	ReferenceId        string            `json:"reference_id,omitempty"`
	ReferenceType      string            `json:"reference_type,omitempty"`

	// Metadata and tracking
	Metadata           map[string]string `json:"metadata,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// UsageRecordListResponse represents a response containing a list of usage records
type UsageRecordListResponse struct {
	Items      []UsageRecordResponse `json:"items"`
	TotalCount int                   `json:"total_count"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
}

// UsageSummaryResponse represents a response containing a summary of usage
type UsageSummaryResponse struct {
	SubscriptionId     string                      `json:"subscription_id"`
	SubscriptionItemId string                      `json:"subscription_item_id"`
	BillingPeriod      string                      `json:"billing_period"`
	UsageType          entities.UsageType          `json:"usage_type"`
	UnitType           entities.UnitType           `json:"unit_type,omitempty"`
	AggregationType    entities.AggregationType    `json:"aggregation_type,omitempty"`
	TotalQuantity      float64                     `json:"total_quantity,omitempty"`
	TotalAmount        int64                       `json:"total_amount"`
	Details            map[string]interface{}      `json:"details,omitempty"`
}

// FromUsageRecord converts a usage record entity to a response
func FromUsageRecord(record entities.UsageRecord) UsageRecordResponse {
	return UsageRecordResponse{
		OrgId:              record.OrgId,
		Id:                 record.Id,
		SubscriptionId:     record.SubscriptionId,
		SubscriptionItemId: record.SubscriptionItemId,
		CustomerId:         record.CustomerId,
		PriceId:            record.PriceId,
		UsageType:          record.UsageType,
		UnitType:           record.UnitType,
		AggregationType:    record.AggregationType,
		Quantity:           record.Quantity,
		UnitPrice:          record.UnitPrice,
		TransactionValue:   record.TransactionValue,
		PercentageRate:     record.PercentageRate,
		CalculatedFee:      record.CalculatedFee,
		FixedFee:           record.FixedFee,
		TotalAmount:        record.TotalAmount,
		UsageDate:          record.UsageDate,
		BillingPeriod:      record.BillingPeriod,
		Processed:          record.Processed,
		ProcessedAt:        record.ProcessedAt,
		InvoiceId:          record.InvoiceId,
		ReferenceId:        record.ReferenceId,
		ReferenceType:      record.ReferenceType,
		Metadata:           record.Metadata,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}
}

// FromUsageRecords converts a slice of usage record entities to a response
func FromUsageRecords(records []entities.UsageRecord, totalCount, page, pageSize int) UsageRecordListResponse {
	var response UsageRecordListResponse
	response.Items = make([]UsageRecordResponse, len(records))
	for i, record := range records {
		response.Items[i] = FromUsageRecord(record)
	}
	response.TotalCount = totalCount
	response.Page = page
	response.PageSize = pageSize
	return response
}
