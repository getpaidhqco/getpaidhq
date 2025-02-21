package request

import "payloop/internal/domain/entities/prices"

// CreateProductRequest represents the request to create a product
type CreateProductRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	Variants    []CreateProductVariantRequest
}

type CreateProductVariantRequest struct {
	Name   string                      `json:"name" binding:"required"`
	Prices []CreateProductPriceRequest `json:"prices"`
}

type CreateProductPriceRequest struct {
	Category           prices.PriceCategory   `json:"category"  binding:"required"`
	Scheme             prices.PriceScheme     `json:"scheme"  binding:"required"`
	Cycles             int                    `json:"cycles"`
	Currency           string                 `json:"currency"  binding:"required"`
	UnitPrice          int64                  `json:"unit_price"  binding:"required"`
	MinPrice           int64                  `json:"min_price"`
	SuggestedPrice     int64                  `json:"suggested_price"`
	BillingInterval    prices.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	TrialInterval      prices.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty"`
	TaxCode            string                 `json:"tax_code"`
	Metadata           map[string]string      `json:"metadata"`
}
