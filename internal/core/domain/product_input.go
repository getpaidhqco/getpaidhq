package domain

// CreateProductInput is the input for creating a product with variants and prices.
type CreateProductInput struct {
	Name        string                        `json:"name" binding:"required"`
	Description string                        `json:"description"`
	Metadata    map[string]string             `json:"metadata"`
	Variants    []CreateProductVariantInput    `json:"variants" binding:"required,dive"`
}

// UpdateProductInput is the input for updating a product.
type UpdateProductInput struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

// CreateProductVariantInput is a variant within a CreateProductInput.
type CreateProductVariantInput struct {
	Name        string                      `json:"name" binding:"required"`
	Description string                      `json:"description"`
	Metadata    map[string]string           `json:"metadata"`
	Prices      []CreateProductPriceInput   `json:"prices" binding:"required,dive"`
}

// CreateProductPriceInput is a price within a CreateProductVariantInput.
type CreateProductPriceInput struct {
	Label              string          `json:"label" binding:"omitempty,min=1,max=255"`
	Category           PriceCategory   `json:"category" binding:"required"`
	Scheme             PriceScheme     `json:"scheme" binding:"required"`
	Cycles             int             `json:"cycles" binding:"omitempty,gt=0"`
	Currency           Currency        `json:"currency" binding:"required"`
	UnitPrice          int64           `json:"unit_price" binding:"required,gte=0"`
	MinPrice           int64           `json:"min_price" binding:"omitempty,gte=0"`
	SuggestedPrice     int64           `json:"suggested_price" binding:"omitempty,gte=0"`
	BillingInterval    BillingInterval `json:"billing_interval"`
	BillingIntervalQty int             `json:"billing_interval_qty"`
	TrialInterval      BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int             `json:"trial_interval_qty"`
	TaxCode            string          `json:"tax_code"`
	Metadata           map[string]string `json:"metadata"`
}

// UpdateVariantInput is the input for updating a variant.
type UpdateVariantInput struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}
