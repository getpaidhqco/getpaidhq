package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type PaymentMethodRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error)
	Create(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error)
}
