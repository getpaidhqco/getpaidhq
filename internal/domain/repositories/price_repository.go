package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type PriceRepository interface {
	Create(ctx context.Context, entity entities.Price) (entities.Price, error)
	FindById(ctx context.Context, orgId string, id string) (entities.Price, error)
}
