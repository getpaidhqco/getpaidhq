package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type UsageRecord struct {
	OrgId              string      `json:"org_id"`
	Id                 string      `json:"id"`
	SubscriptionId     string      `json:"subscription_id"`
	SubscriptionItemId string      `json:"subscription_item_id"`
	CustomerId         string      `json:"customer_id"`

	// Link to price configuration
	PriceId            string      `json:"price_id"`

	// Usage identification
	UsageType          string      `json:"usage_type"`

	// Unit-based usage
	Quantity           pgtype.Numeric `json:"quantity"`
	UnitPrice          pgtype.Int8    `json:"unit_price"`

	// Percentage-based usage
	TransactionValue   pgtype.Int8    `json:"transaction_value"`
	PercentageRate     pgtype.Numeric `json:"percentage_rate"`
	CalculatedFee      pgtype.Int8    `json:"calculated_fee"`

	// Hybrid pricing
	FixedFee           pgtype.Int8    `json:"fixed_fee"`

	// Final billing amount
	TotalAmount        int64          `json:"total_amount"`

	// Time tracking
	UsageDate          pgtype.Timestamp `json:"usage_date"`
	BillingPeriod      string           `json:"billing_period"`

	// Processing status
	Processed          bool             `json:"processed"`
	ProcessedAt        pgtype.Timestamp `json:"processed_at"`
	InvoiceId          pgtype.Text      `json:"invoice_id"`

	// External references
	ReferenceId        pgtype.Text      `json:"reference_id"`
	ReferenceType      pgtype.Text      `json:"reference_type"`

	// Metadata and tracking
	Metadata           map[string]string `json:"metadata"`
	CreatedAt          pgtype.Timestamp  `json:"created_at"`
	UpdatedAt          pgtype.Timestamp  `json:"updated_at"`
}

func (u *UsageRecord) ToEntity() entities.UsageRecord {
	var quantity float64
	if u.Quantity.Valid {
		val, _ := u.Quantity.Value()
		if f, ok := val.(float64); ok {
			quantity = f
		}
	}

	var percentageRate float64
	if u.PercentageRate.Valid {
		val, _ := u.PercentageRate.Value()
		if f, ok := val.(float64); ok {
			percentageRate = f
		}
	}

	return entities.UsageRecord{
		OrgId:              u.OrgId,
		Id:                 u.Id,
		SubscriptionId:     u.SubscriptionId,
		SubscriptionItemId: u.SubscriptionItemId,
		CustomerId:         u.CustomerId,
		PriceId:            u.PriceId,
		UsageType:          u.UsageType,
		Quantity:           quantity,
		UnitPrice:          u.UnitPrice.Int64,
		TransactionValue:   u.TransactionValue.Int64,
		PercentageRate:     percentageRate,
		CalculatedFee:      u.CalculatedFee.Int64,
		FixedFee:           u.FixedFee.Int64,
		TotalAmount:        u.TotalAmount,
		UsageDate:          u.UsageDate.Time,
		BillingPeriod:      u.BillingPeriod,
		Processed:          u.Processed,
		ProcessedAt:        u.ProcessedAt.Time,
		InvoiceId:          u.InvoiceId.String,
		ReferenceId:        u.ReferenceId.String,
		ReferenceType:      u.ReferenceType.String,
		Metadata:           u.Metadata,
		CreatedAt:          u.CreatedAt.Time,
		UpdatedAt:          u.UpdatedAt.Time,
	}
}
