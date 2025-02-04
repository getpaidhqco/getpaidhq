package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type PaymentRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Payment, error)
	Create(ctx context.Context, entity entities.Payment) (entities.Payment, error)
}
