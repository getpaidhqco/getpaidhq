package service

import "getpaidhq/internal/core/domain"

// ProductDetails is the composed read model for "show me a product" queries.
type ProductDetails struct {
	Product  domain.Product
	Variants []VariantDetails
}

// VariantDetails is the per-variant composition used inside ProductDetails,
// and a top-level read model in its own right (variant has its own GET).
type VariantDetails struct {
	Variant domain.Variant
	Prices  []domain.Price
}
