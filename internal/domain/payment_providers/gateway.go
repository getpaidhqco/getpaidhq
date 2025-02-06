package payment_providers

import (
	"context"
	cart "github.com/mdwt/payloop-cart"
	"payloop/internal/domain/entities"
	"time"
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

type PaymentWebhookType string

const (
	PaymentSuccess PaymentWebhookType = "payment.success"
)

type PaymentMethod struct {
	PspId       string `json:"psp_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	IsRecurring bool   `json:"is_recurring"`
	// Token is the payment method token that can be used for future payments
	Token string `json:"token"`
}

type Payment struct {
	Currency  string    `json:"currency"`
	Reference string    `json:"reference"`
	Amount    int       `json:"amount"`
	PaidAt    time.Time `json:"paid_at"`
}

type PaymentWebhookContext struct {
	Type          PaymentWebhookType `json:"type"`
	OrgId         string             `json:"org_id"`
	OrderId       string             `json:"order_id"`
	Psp           string             `json:"psp"`
	Status        string             `json:"status"`
	Payment       Payment            `json:"payment"`
	PaymentMethod PaymentMethod      `json:"payment_method"`
	RawData       []byte             `json:"raw_data"`
}
