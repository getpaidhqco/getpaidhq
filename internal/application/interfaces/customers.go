package interfaces

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type CreatePaymentMethodInput struct {
	request.CreatePaymentMethodRequest
	OrgId      string
	CustomerId string
}
type UpdatePaymentMethodInput struct {
	request.UpdatePaymentMethodRequest
	OrgId           string
	PaymentMethodId string
	CustomerId      string
}

type CustomerService interface {
	GetPaymentMethod(ctx context.Context, orgId string, id string) (entities.PaymentMethod, error)
	Create(ctx context.Context, orgId string, customerRequest request.CreateCustomerRequest) (entities.Customer, error)
	CreatePaymentMethod(ctx context.Context, orgId string, input CreatePaymentMethodInput) (entities.PaymentMethod, error)
	UpdatePaymentMethod(ctx context.Context, orgId string, input UpdatePaymentMethodInput) (entities.PaymentMethod, error)
	Get(ctx context.Context, orgId string, id string) (entities.Customer, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Customer, int, error)
}
