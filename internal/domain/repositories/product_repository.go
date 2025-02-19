package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type ProductRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Product, error)
	Create(ctx context.Context, product entities.Product) (entities.Product, error)
	Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.Product, int, error)
}
