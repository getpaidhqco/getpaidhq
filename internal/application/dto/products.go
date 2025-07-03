package dto

import (
	"payloop/internal/domain/entities/prices"
)

// CreateProductInput represents the input for creating a product
type CreateProductInput struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Metadata    map[string]string           `json:"metadata"`
	Variants    []CreateProductVariantInput `json:"variants"`
}

// CreateProductVariantInput represents the input for creating a product variant
type CreateProductVariantInput struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Metadata    map[string]string         `json:"metadata"`
	Prices      []CreateProductPriceInput `json:"prices"`
}

// CreateProductPriceInput represents the input for creating a product price
type CreateProductPriceInput struct {
	Label              string                 `json:"label"`
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
	UsageType          string                 `json:"usage_type"`
	UnitType           string                 `json:"unit_type"`
	AggregationType    string                 `json:"aggregation_type"`
	PercentageRate     float64                `json:"percentage_rate"`
	FixedFee           int64                  `json:"fixed_fee"`
	OverageUnitPrice   int64                  `json:"overage_unit_price"`
	IncludedUsage      int64                  `json:"included_usage"`
	UsageLimit         int64                  `json:"usage_limit"`

	// Tier configuration
	Tiers              []CreatePriceTierInput `json:"tiers"`

	Metadata           map[string]string      `json:"metadata"`
}

// CreatePriceTierInput represents the input for creating a price tier
type CreatePriceTierInput struct {
	Tier        int    `json:"tier"`
	FromQty     int64  `json:"from_qty"`
	ToQty       int64  `json:"to_qty"`
	UnitPrice   int64  `json:"unit_price"`
	Description string `json:"description"`
}

// UpdateProductInput represents the input for updating a product
type UpdateProductInput struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

// CreateVariantInput represents the input for creating a variant via MCP
type CreateVariantInput struct {
	ProductID   string            `json:"product_id" jsonschema:"required,description=Product ID to create variant for"`
	Name        string            `json:"name" jsonschema:"required,description=Variant name"`
	Description string            `json:"description" jsonschema:"description=Variant description"`
	Metadata    map[string]string `json:"metadata" jsonschema:"description=Variant metadata"`
}

// UpdateVariantInput represents the input for updating a variant via MCP
type UpdateVariantInput struct {
	VariantID   string            `json:"variant_id" jsonschema:"required,description=Variant ID to update"`
	Name        string            `json:"name" jsonschema:"description=Updated variant name"`
	Description string            `json:"description" jsonschema:"description=Updated variant description"`
	Metadata    map[string]string `json:"metadata" jsonschema:"description=Updated variant metadata"`
}

// ListVariantsInput represents the input for listing variants
type ListVariantsInput struct {
	ProductID string `json:"product_id" jsonschema:"required,description=Product ID to list variants for"`
	Page      int    `json:"page" jsonschema:"description=Page number for pagination"`
	Limit     int    `json:"limit" jsonschema:"description=Number of items per page"`
}

// CreatePriceInput represents the input for creating a price via MCP
type CreatePriceInput struct {
	VariantId          string                 `json:"variant_id" jsonschema:"required,description=Variant ID this price belongs to"`
	Category           prices.PriceCategory   `json:"category" jsonschema:"required,enum=one_time,enum=subscription,enum=usage,enum=hybrid,enum=free,enum=variable,description=Price category"`
	Label              string                 `json:"label" jsonschema:"required,description=Human-readable price label"`
	Scheme             prices.PriceScheme     `json:"scheme" jsonschema:"required,enum=flat_rate,enum=per_unit,enum=tiered,enum=volume,description=Pricing scheme"`
	Cycles             int                    `json:"cycles" jsonschema:"description=Number of billing cycles (0 for unlimited)"`
	Currency           string                 `json:"currency" jsonschema:"required,description=Three-letter currency code (e.g. USD)"`
	UnitPrice          int64                  `json:"unit_price" jsonschema:"required,description=Price per unit in smallest currency unit (e.g. cents)"`
	MinPrice           int64                  `json:"min_price" jsonschema:"description=Minimum price in smallest currency unit"`
	SuggestedPrice     int64                  `json:"suggested_price" jsonschema:"description=Suggested price in smallest currency unit"`
	BillingInterval    prices.BillingInterval `json:"billing_interval" jsonschema:"required,enum=none,enum=day,enum=week,enum=month,enum=quarter,enum=year,description=Billing interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty" jsonschema:"description=Number of billing intervals"`
	TrialInterval      prices.BillingInterval `json:"trial_interval" jsonschema:"enum=none,enum=day,enum=week,enum=month,enum=quarter,enum=year,description=Trial period interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty" jsonschema:"description=Number of trial intervals"`
	TaxCode            string                 `json:"tax_code" jsonschema:"description=Tax classification code"`

	// Usage-based billing fields
	HasUsage           bool                   `json:"has_usage" jsonschema:"description=Whether this price includes usage-based billing"`
	UsageType          string                 `json:"usage_type" jsonschema:"description=Type of usage (api_calls, data_transfer, etc.)"`
	UnitType           string                 `json:"unit_type" jsonschema:"description=Unit of measurement (requests, GB, etc.)"`
	AggregationType    string                 `json:"aggregation_type" jsonschema:"enum=sum,enum=max,enum=last_during_period,description=How to aggregate usage records"`
	PercentageRate     float64                `json:"percentage_rate" jsonschema:"description=Percentage rate for transaction fees"`
	FixedFee           int64                  `json:"fixed_fee" jsonschema:"description=Fixed fee per transaction in smallest currency unit"`
	OverageUnitPrice   int64                  `json:"overage_unit_price" jsonschema:"description=Price per unit over included usage"`
	IncludedUsage      int64                  `json:"included_usage" jsonschema:"description=Amount of usage included in base price"`
	UsageLimit         int64                  `json:"usage_limit" jsonschema:"description=Maximum allowed usage"`

	// Tier configuration
	Tiers              []CreatePriceTierInput `json:"tiers" jsonschema:"description=Pricing tiers for tiered/volume pricing"`

	Metadata           map[string]string      `json:"metadata" jsonschema:"description=Price metadata"`
}

// UpdatePriceInput represents the input for updating a price via MCP
type UpdatePriceInput struct {
	PriceID string `json:"price_id" jsonschema:"required,description=Price ID to update"`
	Label              string                 `json:"label" jsonschema:"description=Human-readable price label"`
	Category           prices.PriceCategory   `json:"category" jsonschema:"enum=one_time,enum=subscription,enum=usage,enum=hybrid,enum=free,enum=variable,description=Price category"`
	Scheme             prices.PriceScheme     `json:"scheme" jsonschema:"enum=flat_rate,enum=per_unit,enum=tiered,enum=volume,description=Pricing scheme"`
	Cycles             int                    `json:"cycles" jsonschema:"description=Number of billing cycles (0 for unlimited)"`
	Currency           string                 `json:"currency" jsonschema:"description=Three-letter currency code (e.g. USD)"`
	UnitPrice          int64                  `json:"unit_price" jsonschema:"description=Price per unit in smallest currency unit (e.g. cents)"`
	MinPrice           int64                  `json:"min_price" jsonschema:"description=Minimum price in smallest currency unit"`
	SuggestedPrice     int64                  `json:"suggested_price" jsonschema:"description=Suggested price in smallest currency unit"`
	BillingInterval    prices.BillingInterval `json:"billing_interval" jsonschema:"enum=none,enum=day,enum=week,enum=month,enum=quarter,enum=year,description=Billing interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty" jsonschema:"description=Number of billing intervals"`
	TrialInterval      prices.BillingInterval `json:"trial_interval" jsonschema:"enum=none,enum=day,enum=week,enum=month,enum=quarter,enum=year,description=Trial period interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty" jsonschema:"description=Number of trial intervals"`
	TaxCode            string                 `json:"tax_code" jsonschema:"description=Tax classification code"`

	// Usage-based billing fields
	HasUsage           bool                   `json:"has_usage" jsonschema:"description=Whether this price includes usage-based billing"`
	UsageType          string                 `json:"usage_type" jsonschema:"description=Type of usage (api_calls, data_transfer, etc.)"`
	UnitType           string                 `json:"unit_type" jsonschema:"description=Unit of measurement (requests, GB, etc.)"`
	AggregationType    string                 `json:"aggregation_type" jsonschema:"enum=sum,enum=max,enum=last_during_period,description=How to aggregate usage records"`
	PercentageRate     float64                `json:"percentage_rate" jsonschema:"description=Percentage rate for transaction fees"`
	FixedFee           int64                  `json:"fixed_fee" jsonschema:"description=Fixed fee per transaction in smallest currency unit"`
	OverageUnitPrice   int64                  `json:"overage_unit_price" jsonschema:"description=Price per unit over included usage"`
	IncludedUsage      int64                  `json:"included_usage" jsonschema:"description=Amount of usage included in base price"`
	UsageLimit         int64                  `json:"usage_limit" jsonschema:"description=Maximum allowed usage"`

	// Tier configuration
	Tiers              []CreatePriceTierInput `json:"tiers" jsonschema:"description=Pricing tiers for tiered/volume pricing"`

	Metadata           map[string]string      `json:"metadata" jsonschema:"description=Price metadata"`
}

// ListPricesInput represents the input for listing prices
type ListPricesInput struct {
	VariantId string `json:"variant_id" jsonschema:"required,description=Variant ID to list prices for"`
	Page      int    `json:"page" jsonschema:"description=Page number for pagination"`
	Limit     int    `json:"limit" jsonschema:"description=Number of items per page"`
}

// ProductListFilters represents filters for listing products
type ProductListFilters struct {
	Page   int    `json:"page" jsonschema:"description=Page number for pagination"`
	Limit  int    `json:"limit" jsonschema:"description=Number of items per page"`
	Active *bool  `json:"active" jsonschema:"description=Filter by active status"`
	Search string `json:"search" jsonschema:"description=Search term for product name or description"`
}

// VariantListFilters represents filters for listing variants
type VariantListFilters struct {
	ProductID string `json:"product_id" jsonschema:"required,description=Product ID to list variants for"`
	Page      int    `json:"page" jsonschema:"description=Page number for pagination"`
	Limit     int    `json:"limit" jsonschema:"description=Number of items per page"`
	Search    string `json:"search" jsonschema:"description=Search term for variant name"`
}

// PriceListFilters represents filters for listing prices
type PriceListFilters struct {
	VariantId string `json:"variant_id" jsonschema:"description=Variant ID to filter prices by"`
	ProductId string `json:"product_id" jsonschema:"description=Product ID to filter prices by"`
	Page      int    `json:"page" jsonschema:"description=Page number for pagination"`
	Limit     int    `json:"limit" jsonschema:"description=Number of items per page"`
	Active    *bool  `json:"active" jsonschema:"description=Filter by active status"`
	Currency  string `json:"currency" jsonschema:"description=Filter by currency code"`
}
