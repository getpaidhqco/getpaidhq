package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type CustomerRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Customer, error)
	FindByEmail(ctx context.Context, orgId string, email string) (entities.Customer, error)
	Create(ctx context.Context, entity entities.Customer) (entities.Customer, error)
	Update(ctx context.Context, entity entities.Customer) (entities.Customer, error)
	CreatePaymentMethod(ctx context.Context, entity entities.PaymentMethod) (entities.PaymentMethod, error)
	FindPaymentMethodById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error)
}
