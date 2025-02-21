package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type VariantRepository interface {
	Create(ctx context.Context, entity entities.Variant) (entities.Variant, error)
	FindById(ctx context.Context, orgId string, id string) (entities.Variant, error)
}
