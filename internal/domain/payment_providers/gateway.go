package payment_providers

import (
	"context"
	cart "github.com/mdwt/payloop-cart"
	"payloop/internal/domain/entities"
)

type Gateway interface {
	InitPayment(ctx context.Context, input InitPaymentCommand) (InitPaymentResponse, error)
	ValidateWebhook(ctx context.Context, data []byte) error
	ParseWebhook(ctx context.Context, data []byte) (PaymentWebhookContext, error)
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

type PaymentWebhookContext struct {
	OrgId   string `json:"org_id"`
	OrderId string `json:"order_id"`
	Psp     string `json:"psp"`
	Status  string `json:"status"`
	RawData []byte `json:"raw_data"`
}
