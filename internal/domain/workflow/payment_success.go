package workflow

import (
	"context"
	"payloop/internal/domain/entities"
)

type PaymentSuccessWorkflow interface {
	CompleteOrder(ctx context.Context, order entities.Order) (entities.Order, error)
}
