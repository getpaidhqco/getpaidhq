package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type CustomerRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Customer, error)
	FindByEmail(ctx context.Context, orgId string, email string) (entities.Customer, error)
	Create(ctx context.Context, entity entities.Customer) (entities.Customer, error)
	Update(ctx context.Context, entity entities.Customer) (entities.Customer, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Customer, int, error)

	FindPaymentMethodById(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error)
	AddToCohort(ctx context.Context, orgId string, customerId string, cohortId string, cohortValue string) (entities.Customer, error)
}
