package repositories

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/orders"
)

type PriceRepository interface {
	FindById(ctx context.Context, accountId string, id string) (entities.Order, error)
	Create(ctx context.Context, input orders.CreateOrderRow) (entities.Order, error)
}
