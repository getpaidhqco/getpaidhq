package request

import "payloop/internal/domain/entities/prices"

type CreatePriceRequest struct {
	VariantId          string                 `json:"variant_id" binding:"required"`
	Category           prices.PriceCategory   `json:"category" binding:"required,oneof=one_time subscription free variable"`
	Scheme             prices.PriceScheme     `json:"scheme" binding:"required,oneof=fixed tiered volume graduated"`
	Cycles             int                    `json:"cycles" binding:"omitempty,gt=0"`
	Label              string                 `json:"label"`
	Currency           string                 `json:"currency" binding:"required,iso4217"`
	UnitPrice          int64                  `json:"unit_price" binding:"required,gte=0"`
	MinPrice           int64                  `json:"min_price" binding:"omitempty,gte=0"`
	SuggestedPrice     int64                  `json:"suggested_price" binding:"omitempty,gte=0"`
	BillingInterval    prices.BillingInterval `json:"billing_interval" binding:"omitempty,oneof=none minute hour day week month year"`
	BillingIntervalQty int                    `json:"billing_interval_qty" binding:"omitempty,gt=0,lte=999"`
	TrialInterval      prices.BillingInterval `json:"trial_interval" binding:"omitempty,oneof=none minute hour day week month year"`
	TrialIntervalQty   int                    `json:"trial_interval_qty" binding:"omitempty,gt=0,lte=999"`
	TaxCode            string                 `json:"tax_code" binding:"omitempty,alphanum"`

	// Usage-based billing fields
	HasUsage           bool                   `json:"has_usage" binding:"omitempty"`
	UsageType          string                 `json:"usage_type" binding:"omitempty,oneof=metered licensed"`
	UnitType           string                 `json:"unit_type" binding:"omitempty"`
	AggregationType    string                 `json:"aggregation_type" binding:"omitempty,oneof=sum max average last_during_period"`
	PercentageRate     float64                `json:"percentage_rate" binding:"omitempty,gte=0"`
	FixedFee           int64                  `json:"fixed_fee" binding:"omitempty,gte=0"`
	IncludedUsage      int64                  `json:"included_usage" binding:"omitempty,gte=0"`
	UsageLimit         int64                  `json:"usage_limit" binding:"omitempty,gte=0"`

	// Tier configuration
	Tiers              []CreatePriceTierRequest `json:"tiers" binding:"omitempty,dive"`

	Metadata           map[string]string      `json:"metadata"`
}
