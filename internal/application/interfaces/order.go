package interfaces

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
)

type OrderService interface {
	CreateOrder(ctx context.Context, input orders.CreateOrderInput) (orders.CreateOrderResponse, error)
	CompleteOrder(ctx context.Context, input orders.CompleteOrderInput) (entities.Order, error)
	ListOrderSubscriptions(ctx context.Context, orgId string, id string) ([]entities.Subscription, error)
}

type OrderWorkflowService interface {
	CompleteCheckoutSession(ctx context.Context, input orders.CompleteCheckoutSessionInput) (entities.Order, error)
}
