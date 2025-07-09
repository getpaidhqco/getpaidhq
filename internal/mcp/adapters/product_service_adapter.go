package adapters

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities"
)

// ProductServiceAdapter adapts the existing ProductService to our interface
type ProductServiceAdapter struct {
	productService services.ProductService
}

// NewProductServiceAdapter creates a new adapter for the product service
func NewProductServiceAdapter(productService services.ProductService) *ProductServiceAdapter {
	return &ProductServiceAdapter{
		productService: productService,
	}
}

// Product operations
func (a *ProductServiceAdapter) CreateProduct(ctx context.Context, orgId string, input dto.CreateProductInput) (entities.Product, error) {
	return a.productService.CreateProduct(ctx, orgId, input)
}

func (a *ProductServiceAdapter) UpdateProduct(ctx context.Context, orgId string, productId string, input dto.UpdateProductInput) (entities.Product, error) {
	return a.productService.UpdateProduct(ctx, orgId, productId, input)
}

func (a *ProductServiceAdapter) FindById(ctx context.Context, orgId string, id string) (entities.Product, error) {
	return a.productService.FindById(ctx, orgId, id)
}

func (a *ProductServiceAdapter) List(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Product, int, error) {
	// Convert dto.Pagination to request.Pagination
	reqPagination := request.Pagination{
		Page:   pagination.Page,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}
	return a.productService.List(ctx, orgId, reqPagination)
}

func (a *ProductServiceAdapter) DeleteProduct(ctx context.Context, orgId string, id string) error {
	return a.productService.DeleteProduct(ctx, orgId, id)
}

// Variant operations
func (a *ProductServiceAdapter) CreateVariant(ctx context.Context, orgId string, productId string, input dto.CreateVariantInput) (entities.Variant, error) {
	// Convert dto to request format
	req := request.CreateVariantRequest{
		Name:        input.Name,
		Description: input.Description,
		Metadata:    input.Metadata,
	}
	return a.productService.CreateVariant(ctx, orgId, productId, req)
}

func (a *ProductServiceAdapter) UpdateVariant(ctx context.Context, orgId string, variantId string, input dto.UpdateVariantInput) (entities.Variant, error) {
	// Convert dto to request format
	req := request.UpdateVariantRequest{
		Name:        input.Name,
		Description: input.Description,
		Metadata:    input.Metadata,
	}
	return a.productService.UpdateVariant(ctx, orgId, variantId, req)
}

func (a *ProductServiceAdapter) GetVariant(ctx context.Context, orgId string, id string) (entities.Variant, error) {
	return a.productService.GetVariant(ctx, orgId, id)
}

func (a *ProductServiceAdapter) ListVariants(ctx context.Context, orgId string, productId string, pagination dto.Pagination) ([]entities.Variant, int, error) {
	// Convert dto.Pagination to request.Pagination
	reqPagination := request.Pagination{
		Page:   pagination.Page,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}
	return a.productService.ListVariants(ctx, orgId, productId, reqPagination)
}

func (a *ProductServiceAdapter) DeleteVariant(ctx context.Context, orgId string, id string) error {
	return a.productService.DeleteVariant(ctx, orgId, id)
}

// Price operations
func (a *ProductServiceAdapter) CreateProductPrice(ctx context.Context, input dto.CreatePriceInput) (entities.Price, error) {
	// Convert dto to entities format that the service expects
	entitiesInput := entities.CreatePriceInput{
		OrgId:              "", // Will be set from context in the service
		VariantId:          input.VariantId,
		Category:           input.Category,
		Label:              input.Label,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           input.Currency,
		UnitPrice:          input.UnitPrice,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		HasUsage:           input.HasUsage,
		MeterId:            input.MeterId,
		PercentageRate:     input.PercentageRate,
		FixedFee:           input.FixedFee,
		OverageUnitPrice:   input.OverageUnitPrice,
		IncludedUsage:      input.IncludedUsage,
		UsageLimit:         input.UsageLimit,
		Metadata:           input.Metadata,
	}

	// Convert tiers
	var tiers []entities.CreatePriceTierInput
	for _, tier := range input.Tiers {
		fromQty := int(tier.FromQty)
		var toQty *int
		if tier.ToQty > 0 {
			toQtyInt := int(tier.ToQty)
			toQty = &toQtyInt
		}

		tiers = append(tiers, entities.CreatePriceTierInput{
			Tier:        tier.Tier,
			FromQty:     fromQty,
			ToQty:       toQty,
			UnitPrice:   tier.UnitPrice,
			Description: tier.Description,
		})
	}
	entitiesInput.Tiers = tiers

	return a.productService.CreateProductPrice(ctx, entitiesInput)
}

func (a *ProductServiceAdapter) UpdatePrice(ctx context.Context, orgId string, priceId string, input dto.UpdatePriceInput) (entities.Price, error) {
	// Convert dto to entities format that the service expects
	entitiesInput := entities.CreatePriceInput{
		OrgId:              orgId,
		VariantId:          "", // Not needed for updates
		Category:           input.Category,
		Label:              input.Label,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           input.Currency,
		UnitPrice:          input.UnitPrice,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		HasUsage:           input.HasUsage,
		MeterId:            input.MeterId,
		PercentageRate:     input.PercentageRate,
		FixedFee:           input.FixedFee,
		OverageUnitPrice:   input.OverageUnitPrice,
		IncludedUsage:      input.IncludedUsage,
		UsageLimit:         input.UsageLimit,
		Metadata:           input.Metadata,
	}

	// Convert tiers
	var tiers []entities.CreatePriceTierInput
	for _, tier := range input.Tiers {
		fromQty := int(tier.FromQty)
		var toQty *int
		if tier.ToQty > 0 {
			toQtyInt := int(tier.ToQty)
			toQty = &toQtyInt
		}

		tiers = append(tiers, entities.CreatePriceTierInput{
			Tier:        tier.Tier,
			FromQty:     fromQty,
			ToQty:       toQty,
			UnitPrice:   tier.UnitPrice,
			Description: tier.Description,
		})
	}
	entitiesInput.Tiers = tiers

	return a.productService.UpdatePrice(ctx, orgId, priceId, entitiesInput)
}

func (a *ProductServiceAdapter) GetPrice(ctx context.Context, orgId string, id string) (entities.Price, error) {
	return a.productService.GetPrice(ctx, orgId, id)
}

func (a *ProductServiceAdapter) ListPrices(ctx context.Context, orgId string, variantId string, pagination dto.Pagination) ([]entities.Price, int, error) {
	// Convert dto.Pagination to request.Pagination
	reqPagination := request.Pagination{
		Page:   pagination.Page,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}
	return a.productService.ListPrices(ctx, orgId, variantId, reqPagination)
}

func (a *ProductServiceAdapter) DeletePrice(ctx context.Context, orgId string, id string) error {
	return a.productService.DeletePrice(ctx, orgId, id)
}
