package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type OrderItemRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.OrderItem, error)
	Create(ctx context.Context, entity entities.OrderItem) (entities.OrderItem, error)
	Update(ctx context.Context, orderItem entities.OrderItem) (entities.OrderItem, error)
	FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.OrderItem, error)
}
