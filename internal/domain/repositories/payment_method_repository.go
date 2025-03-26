package repositories

import (
	"context"
	"payloop/internal/domain/entities"
	"time"
)

type PaymentMethodRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error)
	Create(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error)
	Update(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error)

	FindExpiringPaymentMethods(ctx context.Context, expiry time.Time) ([]entities.PaymentMethod, error)
}
