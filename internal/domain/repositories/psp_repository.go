package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type PspRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.PaymentServiceProvider, error)
	Create(ctx context.Context, input entities.PaymentServiceProvider) (entities.PaymentServiceProvider, error)
}
