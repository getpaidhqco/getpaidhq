package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type PriceRepository interface {
	Create(ctx context.Context, entity entities.Price) (entities.Price, error)
	FindById(ctx context.Context, orgId string, id string) (entities.Price, error)
	FindByVariantId(ctx context.Context, orgId string, variantId string, p request.Pagination) ([]entities.Price, int, error)
	Update(ctx context.Context, entity entities.Price) (entities.Price, error)
	Delete(ctx context.Context, orgId string, id string) error

	// Tier management
	GetPriceTiers(ctx context.Context, orgId string, priceId string) ([]entities.PriceTier, error)
	CreatePriceTiers(ctx context.Context, tiers []entities.PriceTier) error
	UpdatePriceTiers(ctx context.Context, orgId string, priceId string, tiers []entities.PriceTier) error
	DeletePriceTiers(ctx context.Context, orgId string, priceId string) error
}
