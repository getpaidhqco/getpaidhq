package service

// CreateProductInput is the command input for ProductService.Create.
type CreateProductInput struct {
	Name        string
	Description string
	Metadata    map[string]string
	Variants    []CreateProductVariantInput
}

// UpdateProductInput is the command input for ProductService.Update.
type UpdateProductInput struct {
	Name        string
	Description string
	Metadata    map[string]string
}

// CreateProductVariantInput is a variant within a CreateProductInput.
type CreateProductVariantInput struct {
	Name        string
	Description string
	Metadata    map[string]string
	Prices      []CreateProductPriceInput
}
