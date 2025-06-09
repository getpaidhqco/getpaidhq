package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type VariantRepository interface {
	Create(ctx context.Context, entity entities.Variant) (entities.Variant, error)
	FindById(ctx context.Context, orgId string, id string) (entities.Variant, error)
	FindByProductId(ctx context.Context, orgId string, productId string, p request.Pagination) ([]entities.Variant, int, error)
	Update(ctx context.Context, entity entities.Variant) (entities.Variant, error)
	Delete(ctx context.Context, orgId string, id string) error
}
