package payment_providers

import (
	"context"
	cart "github.com/mdwt/payloop-cart"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"time"
)

type Gateway interface {
	InitPayment(ctx context.Context, input InitPaymentCommand) (InitPaymentResponse, error)
	ChargePayment(ctx context.Context, input ChargePaymentCommand) ChargePaymentResponse
}

type WebhookParser interface {
	ValidateWebhook(ctx context.Context, data []byte) error
	ParseWebhook(ctx context.Context, data []byte) (PaymentWebhookContext, error)
}

type ChargePaymentCommand struct {
	OrgId         string
	Amount        int64
	Currency      string
	Reference     string
	PaymentMethod PaymentMethod
	Customer      entities.Customer
}
type InitPaymentCommand struct {
	OrgId    string
	Cart     cart.Cart
	Order    entities.Order
	Customer entities.Customer
}

type ChargePaymentResponse struct {
	Success       bool            `json:"success"`
	Retryable     bool            `json:"retryable"`
	Psp           common.Gateway  `json:"psp"`
	PspId         string          `json:"psp_id"`
	Reference     string          `json:"reference"`
	Currency      common.Currency `json:"currency"`
	AmountCharged int64           `json:"amount_charged"`
	PaymentType   string          `json:"payment_type"`

	PspResponse interface{} `json:"psp_response"`
}

type InitPaymentResponse struct {
	PspResponse interface{}
}

type PaymentWebhookType string

const (
	Noop           PaymentWebhookType = "noop"
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
	Currency    string    `json:"currency"`
	Reference   string    `json:"reference"`
	PspId       string    `json:"psp_id"`
	Amount      int64     `json:"amount"`
	PaidAt      time.Time `json:"paid_at"`
	PspFee      int       `json:"psp_fee"`
	PlatformFee int       `json:"platform_fee"`
}

type Customer struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	PspId     string `json:"psp_id"`
}

type PaymentWebhookContext struct {
	Type          PaymentWebhookType `json:"type"`
	OrgId         string             `json:"org_id"`
	OrderId       string             `json:"order_id"`
	Psp           string             `json:"psp"`
	Status        string             `json:"status"`
	Payment       Payment            `json:"payment"`
	Customer      Customer           `json:"customer"`
	PaymentMethod PaymentMethod      `json:"payment_method"`
	RawData       []byte             `json:"raw_data"`
}
