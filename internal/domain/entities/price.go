package entities

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/lib"
	"time"
)

type Price struct {
	OrgId              string                 `json:"org_id"`
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
	HasUsage           bool                    `json:"has_usage"`
	UsageType          prices.UsageType        `json:"usage_type,omitempty"`
	UnitType           prices.UnitType         `json:"unit_type,omitempty"`
	AggregationType    prices.AggregationType  `json:"aggregation_type,omitempty"`
	PercentageRate     float64                 `json:"percentage_rate,omitempty"`
	FixedFee           int64                   `json:"fixed_fee,omitempty"`
	OverageUnitPrice   int64                   `json:"overage_unit_price,omitempty"`
	IncludedUsage      int64                   `json:"included_usage,omitempty"`
	UsageLimit         int64                   `json:"usage_limit,omitempty"`

	// Tier configuration
	Tiers              []PriceTier             `json:"tiers,omitempty"`

	Metadata           map[string]string       `json:"metadata"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

// Factory function to create a Price with default values
func NewPrice(orgId, variantId string, input CreatePriceInput) Price {

	if input.BillingInterval == "" {
		input.BillingInterval = prices.BillingIntervalNone
	}
	if input.TrialInterval == "" {
		input.TrialInterval = prices.BillingIntervalNone
	}

	return Price{
		OrgId:              orgId,
		Id:                 lib.GenerateId("price"),
		Label:              input.Label,
		VariantId:          variantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           common.Currency(input.Currency),
		UnitPrice:          input.UnitPrice,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,

		// Usage-based billing fields
		HasUsage:           input.HasUsage,
		UsageType:          prices.UsageType(input.UsageType),
		UnitType:           prices.UnitType(input.UnitType),
		AggregationType:    prices.AggregationType(input.AggregationType),
		PercentageRate:     input.PercentageRate,
		FixedFee:           input.FixedFee,
		OverageUnitPrice:   input.OverageUnitPrice,
		IncludedUsage:      input.IncludedUsage,
		UsageLimit:         input.UsageLimit,

		Metadata:           input.Metadata,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
}

type CreatePriceInput struct {
	OrgId              string                 `json:"org_id"`
	Label              string                 `json:"label"`
	VariantId          string                 `json:"variant_id"`
	Category           prices.PriceCategory   `json:"category"`
	Scheme             prices.PriceScheme     `json:"scheme"`
	Cycles             int                    `json:"cycles"`
	Currency           string                 `json:"currency"`
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
	UsageType          string                 `json:"usage_type,omitempty"`
	UnitType           string                 `json:"unit_type,omitempty"`
	AggregationType    string                 `json:"aggregation_type,omitempty"`
	PercentageRate     float64                `json:"percentage_rate,omitempty"`
	FixedFee           int64                  `json:"fixed_fee,omitempty"`
	OverageUnitPrice   int64                  `json:"overage_unit_price,omitempty"`
	IncludedUsage      int64                  `json:"included_usage,omitempty"`
	UsageLimit         int64                  `json:"usage_limit,omitempty"`

	Metadata           map[string]string      `json:"metadata"`
}
