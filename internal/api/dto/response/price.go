package response

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
	"time"
)

type Price struct {
	Id                 string                 `json:"id"`
	VariantId          string                 `json:"variant_id"`
	Label              string                 `json:"label"`
	Category           prices.PriceCategory   `json:"category"`
	Scheme             prices.PriceScheme     `json:"scheme"`
	Cycles             int                    `json:"cycles"`
	Currency           common.Currency        `json:"currency"`
	UnitPrice          int64                  `json:"unit_price"`
	MinPrice           int64                  `json:"min_price"`
	SuggestedPrice     int64                  `json:"suggested_price"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	TrialInterval      prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty"`
	TaxCode            string                 `json:"tax_code"`

	// Usage-based billing fields
	HasUsage           bool                   `json:"has_usage"`
	UsageType          prices.UsageType       `json:"usage_type,omitempty"`
	UnitType           prices.UnitType        `json:"unit_type,omitempty"`
	AggregationType    prices.AggregationType `json:"aggregation_type,omitempty"`
	PercentageRate     float64                `json:"percentage_rate,omitempty"`
	FixedFee           int64                  `json:"fixed_fee,omitempty"`
	IncludedUsage      int64                  `json:"included_usage,omitempty"`
	UsageLimit         int64                  `json:"usage_limit,omitempty"`

	// Tier configuration
	Tiers              []PriceTier            `json:"tiers,omitempty"`

	Metadata           map[string]string      `json:"metadata"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

func NewPriceFromEntity(entity entities.Price) Price {
	return Price{
		Id:                 entity.Id,
		VariantId:          entity.VariantId,
		Category:           entity.Category,
		Scheme:             entity.Scheme,
		Label:              entity.Label,
		Cycles:             entity.Cycles,
		Currency:           entity.Currency,
		UnitPrice:          entity.UnitPrice,
		MinPrice:           entity.MinPrice,
		SuggestedPrice:     entity.SuggestedPrice,
		BillingInterval:    entity.BillingInterval,
		BillingIntervalQty: entity.BillingIntervalQty,
		TrialInterval:      entity.TrialInterval,
		TrialIntervalQty:   entity.TrialIntervalQty,
		TaxCode:            entity.TaxCode,

		// Usage-based billing fields
		HasUsage:           entity.HasUsage,
		UsageType:          entity.UsageType,
		UnitType:           entity.UnitType,
		AggregationType:    entity.AggregationType,
		PercentageRate:     entity.PercentageRate,
		FixedFee:           entity.FixedFee,
		IncludedUsage:      entity.IncludedUsage,
		UsageLimit:         entity.UsageLimit,

		// Tier configuration
		Tiers:              NewPriceTiersFromEntities(entity.Tiers),

		Metadata:           entity.Metadata,
		CreatedAt:          entity.CreatedAt,
		UpdatedAt:          entity.UpdatedAt,
	}
}
