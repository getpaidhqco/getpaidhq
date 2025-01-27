package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type PriceRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Price, error)
}
