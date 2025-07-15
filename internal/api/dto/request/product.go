package request

import "payloop/internal/domain/entities/prices"

// CreateProductRequest represents the request to create a product
type CreateProductRequest struct {
	Name        string                        `json:"name" binding:"required"`
	Description string                        `json:"description"`
	Metadata    map[string]string             `json:"metadata"`
	Variants    []CreateProductVariantRequest `json:"variants" binding:"required,dive"`
}

// UpdateProductRequest represents the request to update a product
type UpdateProductRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

// CreateVariantRequest represents the request to create a variant
type CreateVariantRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

// UpdateVariantRequest represents the request to update a variant
type UpdateVariantRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

type CreateProductVariantRequest struct {
	Name        string                      `json:"name" binding:"required"`
	Description string                      `json:"description"`
	Metadata    map[string]string           `json:"metadata"`
	Prices      []CreateProductPriceRequest `json:"prices" binding:"required,dive"`
}

type CreateProductPriceRequest struct {
	Label              string                 `json:"label" binding:"omitempty,min=1,max=255"`
	Category           prices.PriceCategory   `json:"category" binding:"required,oneof=one_time subscription usage hybrid free variable"`
	Scheme             prices.PriceScheme     `json:"scheme" binding:"required,oneof=fixed tiered volume graduated"`
	Cycles             int                    `json:"cycles" binding:"omitempty,gt=0"`
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
	HasUsage           bool                   `json:"has_usage"`
	MeterId            string                 `json:"meter_id" binding:"required_if=HasUsage true,omitempty"`
	PercentageRate     float64                `json:"percentage_rate" binding:"omitempty,gte=0"`
	FixedFee           int64                  `json:"fixed_fee" binding:"omitempty,gte=0"`
	OverageUnitPrice   int64                  `json:"overage_unit_price" binding:"omitempty,gte=0"`
	IncludedUsage      int64                  `json:"included_usage" binding:"omitempty,gte=0"`
	UsageLimit         int64                  `json:"usage_limit" binding:"omitempty,gte=0"`

	// Tier configuration
	Tiers              []CreatePriceTierRequest `json:"tiers" binding:"omitempty,dive"`

	Metadata           map[string]string      `json:"metadata"`
}
