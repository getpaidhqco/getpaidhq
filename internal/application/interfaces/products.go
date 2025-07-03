package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

type ProductService interface {
	// Product operations
	CreateProduct(ctx context.Context, orgId string, input dto.CreateProductInput) (entities.Product, error)
	UpdateProduct(ctx context.Context, orgId string, productId string, input dto.UpdateProductInput) (entities.Product, error)
	FindById(ctx context.Context, orgId string, id string) (entities.Product, error)
	List(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Product, int, error)
	DeleteProduct(ctx context.Context, orgId string, id string) error

	// Variant operations
	CreateVariant(ctx context.Context, orgId string, productId string, input dto.CreateVariantInput) (entities.Variant, error)
	UpdateVariant(ctx context.Context, orgId string, variantId string, input dto.UpdateVariantInput) (entities.Variant, error)
	GetVariant(ctx context.Context, orgId string, id string) (entities.Variant, error)
	ListVariants(ctx context.Context, orgId string, productId string, pagination dto.Pagination) ([]entities.Variant, int, error)
	DeleteVariant(ctx context.Context, orgId string, id string) error

	// Price operations
	CreateProductPrice(ctx context.Context, input dto.CreatePriceInput) (entities.Price, error)
	UpdatePrice(ctx context.Context, orgId string, priceId string, input dto.UpdatePriceInput) (entities.Price, error)
	GetPrice(ctx context.Context, orgId string, id string) (entities.Price, error)
	ListPrices(ctx context.Context, orgId string, variantId string, pagination dto.Pagination) ([]entities.Price, int, error)
	DeletePrice(ctx context.Context, orgId string, id string) error
}