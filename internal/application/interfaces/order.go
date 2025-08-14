package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
)

type OrderService interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Order, error)
	CreateOrder(ctx context.Context, input orders.CreateOrderInput) (orders.CreateOrderResponse, error)
	CompleteOrder(ctx context.Context, input orders.CompleteOrderInput) (entities.Order, error)
	List(ctx context.Context, orgId string, pagination dto.Pagination) ([]entities.Order, int, error)
	ListOrderSubscriptions(ctx context.Context, orgId string, id string) ([]entities.Subscription, error)
}

type OrderWorkflowService interface {
	CompleteCheckoutSession(ctx context.Context, input orders.CompleteCheckoutSessionInput) (entities.Order, error)
}
