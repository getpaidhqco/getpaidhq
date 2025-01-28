package payment_providers

import (
	"context"
	cart "github.com/mdwt/payloop-cart"
	"payloop/internal/domain/entities"
)

type Gateway interface {
	InitPayment(ctx context.Context, input InitPaymentCommand) (InitPaymentResponse, error)
}

type InitPaymentCommand struct {
	OrgId    string
	Cart     cart.Cart
	Order    entities.Order
	Customer entities.Customer
}

type InitPaymentResponse struct {
	PspResponse interface{}
}
