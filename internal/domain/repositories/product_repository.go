package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type ProductRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Product, error)
}
