package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

type CustomerService interface {
	// Customer operations
	Create(ctx context.Context, orgId string, input dto.CreateCustomerInput) (entities.Customer, error)
	Update(ctx context.Context, orgId string, customerId string, input dto.UpdateCustomerInput) (entities.Customer, error)
	Get(ctx context.Context, orgId string, id string) (entities.Customer, error)
	List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Customer], error)

	// Payment method operations
	CreatePaymentMethod(ctx context.Context, orgId string, input dto.CreatePaymentMethodInput) (entities.PaymentMethod, error)
	UpdatePaymentMethod(ctx context.Context, orgId string, paymentMethodId string, input dto.UpdatePaymentMethodInput) (entities.PaymentMethod, error)
	GetPaymentMethod(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error)

	// Secure payment method operations
	GetSecurePaymentMethod(ctx context.Context, orgId string, id string) (entities.SecurePaymentMethod, error)
	CreateSecurePaymentMethod(ctx context.Context, orgId string, input dto.CreatePaymentMethodInput) (entities.SecurePaymentMethod, error)
	UpdateSecurePaymentMethod(ctx context.Context, orgId string, paymentMethodId string, input dto.UpdatePaymentMethodInput) (entities.SecurePaymentMethod, error)
}
