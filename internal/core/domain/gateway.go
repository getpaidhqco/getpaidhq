package domain

import (
	"context"
	"time"
)

type GatewayConfig interface {
	Validate() error
}

type GatewayProvider interface {
	InitPayment(ctx context.Context, input InitPaymentCommand) (InitPaymentResponse, error)
	ChargePayment(ctx context.Context, input ChargePaymentCommand) ChargePaymentResponse
}

type WebhookParser interface {
	ValidateWebhook(ctx context.Context, data []byte) error
	ParseWebhook(ctx context.Context, data []byte) (PaymentWebhookContext, error)
}

type ChargePaymentCommand struct {
	OrgId          string
	OrderId        string
	SubscriptionId string
	Amount         int64
	Currency       Currency
	Reference      string
	PaymentMethod  GatewayPaymentMethod
	Customer       Customer
}

type InitPaymentCommand struct {
	OrgId    string
	Cart     Cart
	Order    Order
	Customer Customer
	Options  map[string]string
}

type ChargePaymentStatus string

const (
	ChargePaymentStatusSuccess ChargePaymentStatus = "Success"
	ChargePaymentStatusPending ChargePaymentStatus = "Pending"
	ChargePaymentStatusError   ChargePaymentStatus = "Error"

	// GatewayError is a generic error relating to comms with the gateway. Common error is a 429 rate exceeded.
	// This is not a user error and should be retried by the platform instead of being seen as a failed payment.
	GatewayError ChargePaymentStatus = "gateway_error"
)

type ChargePaymentResponse struct {
	Status        ChargePaymentStatus `json:"status"`
	Retryable     bool                `json:"retryable"`
	Psp           Gateway             `json:"psp"`
	PspId         string              `json:"psp_id"`
	Reference     string              `json:"reference"`
	Currency      Currency            `json:"currency"`
	AmountCharged int64               `json:"amount_charged"`
	PaymentType   string              `json:"payment_type"`
	ErrorReason   string              `json:"error_reason"`
	ErrorCode     string              `json:"error_code"`

	PspResponse interface{} `json:"psp_response"`
}

type InitPaymentResponse struct {
	PspResponse interface{}
}

type PaymentWebhookType string

const (
	Noop             PaymentWebhookType = "noop"
	PaymentSuccess   PaymentWebhookType = "payment.success"
	PaymentRefunded  PaymentWebhookType = "payment.refunded"
	RecurringSuccess PaymentWebhookType = "recurring.success"
)

type GatewayPaymentMethod struct {
	PspId       string `json:"psp_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	IsRecurring bool   `json:"is_recurring"`
	// Token is the payment method token that can be used for future payments
	Token string `json:"token"`
}

type GatewayPayment struct {
	Currency    Currency      `json:"currency"`
	Reference   string        `json:"reference"`
	PspId       string        `json:"psp_id"`
	Amount      int64         `json:"amount"`
	Status      PaymentStatus `json:"status"`
	PaidAt      time.Time     `json:"paid_at"`
	PspFee      int           `json:"psp_fee"`
	PlatformFee int           `json:"platform_fee"`
}

type GatewayCustomer struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	PspId     string `json:"psp_id"`
}

type PaymentWebhookContext struct {
	Type          PaymentWebhookType   `json:"type"`
	OrgId         string               `json:"org_id"`
	OrderId       string               `json:"order_id"`
	Psp           Gateway              `json:"psp"`
	Status        string               `json:"status"`
	Payment       GatewayPayment       `json:"payment"`
	Customer      GatewayCustomer      `json:"customer"`
	PaymentMethod GatewayPaymentMethod `json:"payment_method"`
	RawData       []byte               `json:"raw_data"`
}
