package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type SubscriptionItem struct {
	OrgId          string      `json:"org_id"`
	Id             string      `json:"id"`
	SubscriptionId string      `json:"subscription_id"`

	// Product/Price reference
	PriceId        string      `json:"price_id"`
	ProductId      pgtype.Text `json:"product_id"`
	VariantId      pgtype.Text `json:"variant_id"`

	// Item details
	Name           string      `json:"name"`
	Description    pgtype.Text `json:"description"`
	Status         string      `json:"status"`

	// Quantity for fixed items
	Quantity       int         `json:"quantity"`

	// Billing
	Amount         pgtype.Int8 `json:"amount"`
	Currency       string      `json:"currency"`

	// Pricing configuration
	PercentageRate pgtype.Float8 `json:"percentage_rate"`
	FixedFee       pgtype.Int8   `json:"fixed_fee"`
	UnitPrice      pgtype.Int8   `json:"unit_price"`

	// Usage configuration
	HasUsage       bool        `json:"has_usage"`
	UsageType      pgtype.Text `json:"usage_type"`
	UnitType       pgtype.Text `json:"unit_type"`
	AggregationType pgtype.Text `json:"aggregation_type"`

	// Metadata
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      pgtype.Timestamp  `json:"created_at"`
	UpdatedAt      pgtype.Timestamp  `json:"updated_at"`
}

func (s *SubscriptionItem) ToEntity() entities.SubscriptionItem {
	return entities.SubscriptionItem{
		OrgId:          s.OrgId,
		Id:             s.Id,
		SubscriptionId: s.SubscriptionId,
		PriceId:        s.PriceId,
		ProductId:      s.ProductId.String,
		VariantId:      s.VariantId.String,
		Name:           s.Name,
		Description:    s.Description.String,
		Status:         entities.SubscriptionItemStatus(s.Status),
		Quantity:       s.Quantity,
		Amount:         s.Amount.Int64,
		Currency:       s.Currency,
		PercentageRate: s.PercentageRate.Float64,
		FixedFee:       s.FixedFee.Int64,
		UnitPrice:      s.UnitPrice.Int64,
		HasUsage:       s.HasUsage,
		UsageType:      entities.UsageType(s.UsageType.String),
		UnitType:       entities.UnitType(s.UnitType.String),
		AggregationType: entities.AggregationType(s.AggregationType.String),
		Metadata:       s.Metadata,
		CreatedAt:      s.CreatedAt.Time,
		UpdatedAt:      s.UpdatedAt.Time,
	}
}
