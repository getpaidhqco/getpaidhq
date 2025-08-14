package repositories

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

type OrderRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Order, error)
	Create(ctx context.Context, entity entities.Order) (entities.Order, error)
	Update(ctx context.Context, entity entities.Order) (entities.Order, error)
	Find(ctx context.Context, orgId string, p dto.Pagination) ([]entities.Order, int, error)
}
