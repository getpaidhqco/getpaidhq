package entities

import (
	"payloop/internal/lib"
	"time"
)

// UsageRecord represents a record of usage for a subscription item
type UsageRecord struct {
	OrgId              string            `json:"org_id"`
	Id                 string            `json:"id"`
	SubscriptionId     string            `json:"subscription_id"`
	SubscriptionItemId string            `json:"subscription_item_id"`
	CustomerId         string            `json:"customer_id"`
	
	// Link to price configuration
	PriceId            string            `json:"price_id"`
	
	// Usage identification
	UsageType          string            `json:"usage_type"`
	
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

// NewUnitUsageRecord creates a new usage record for unit-based usage
func NewUnitUsageRecord(orgId, subscriptionId, subscriptionItemId, customerId, priceId string, quantity float64, unitPrice int64) UsageRecord {
	totalAmount := int64(quantity * float64(unitPrice))
	
	return UsageRecord{
		OrgId:              orgId,
		Id:                 lib.GenerateId("ur"),
		SubscriptionId:     subscriptionId,
		SubscriptionItemId: subscriptionItemId,
		CustomerId:         customerId,
		PriceId:            priceId,
		UsageType:          "unit",
		Quantity:           quantity,
		UnitPrice:          unitPrice,
		TotalAmount:        totalAmount,
		UsageDate:          time.Now().UTC(),
		BillingPeriod:      formatBillingPeriod(time.Now().UTC()),
		Processed:          false,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

// NewPercentageUsageRecord creates a new usage record for percentage-based usage
func NewPercentageUsageRecord(orgId, subscriptionId, subscriptionItemId, customerId, priceId string, transactionValue int64, percentageRate float64) UsageRecord {
	calculatedFee := int64(float64(transactionValue) * percentageRate / 100)
	
	return UsageRecord{
		OrgId:              orgId,
		Id:                 lib.GenerateId("ur"),
		SubscriptionId:     subscriptionId,
		SubscriptionItemId: subscriptionItemId,
		CustomerId:         customerId,
		PriceId:            priceId,
		UsageType:          "percentage",
		TransactionValue:   transactionValue,
		PercentageRate:     percentageRate,
		CalculatedFee:      calculatedFee,
		TotalAmount:        calculatedFee,
		UsageDate:          time.Now().UTC(),
		BillingPeriod:      formatBillingPeriod(time.Now().UTC()),
		Processed:          false,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

// MarkProcessed marks the usage record as processed
func (u *UsageRecord) MarkProcessed(invoiceId string) *UsageRecord {
	u.Processed = true
	u.ProcessedAt = time.Now().UTC()
	u.InvoiceId = invoiceId
	u.UpdatedAt = time.Now().UTC()
	return u
}

// SetMetadata merges the existing metadata with the specified values.
func (u *UsageRecord) SetMetadata(meta map[string]string) *UsageRecord {
	if u.Metadata == nil {
		u.Metadata = make(map[string]string)
	}
	for key, value := range meta {
		u.Metadata[key] = value
	}
	return u
}

// formatBillingPeriod formats the billing period as YYYY-MM
func formatBillingPeriod(date time.Time) string {
	return date.Format("2006-01")
}