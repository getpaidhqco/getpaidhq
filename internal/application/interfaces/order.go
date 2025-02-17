package interfaces

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/payment_providers"
)

type OrderService interface {
	CreateOrderFromCart(ctx context.Context, input orders.CreateOrderInput) (entities.Order, payment_providers.InitPaymentResponse, error)
	CompleteOrder(ctx context.Context, input orders.CompleteOrderCommand) (entities.Order, error)
}
