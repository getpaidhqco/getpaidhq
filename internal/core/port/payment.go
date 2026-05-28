package port

import (
	"context"
	"encoding/json"
	"errors"
	"getpaidhq/internal/core/domain"
	"time"
)

// PaymentGateway is the port for payment service provider operations.
// Named PaymentGateway (not Gateway) to avoid collision with domain.Gateway string type.
type PaymentGateway interface {
	InitPayment(ctx context.Context, input InitPaymentCommand) (InitPaymentResponse, error)
	ChargePayment(ctx context.Context, input ChargePaymentCommand) ChargePaymentResponse
}

// PaymentGatewayConfig validates PSP-specific configuration.
type PaymentGatewayConfig interface {
	Validate() error
}

// WebhookParser validates and parses incoming payment webhooks. The
// `signature` argument carries the PSP-provided HMAC of the raw body
// (e.g. X-Paystack-Signature, Cko-Signature); implementations MUST
// verify it constant-time before considering the body authentic.
type WebhookParser interface {
	ValidateWebhook(ctx context.Context, data []byte, signature string) error
	ParseWebhook(ctx context.Context, data []byte) (PaymentWebhookContext, error)
}

type ChargePaymentCommand struct {
	OrgId          string
	OrderId        string
	SubscriptionId string
	Amount         int64
	Currency       string
	Reference      string
	PaymentMethod  PspPaymentMethod
	Customer       domain.Customer
}

type InitPaymentCommand struct {
	OrgId    string
	Cart     any // cart.Cart - resolved at adapter level
	Order    domain.Order
	Customer domain.Customer
	Options  map[string]string
}

type ChargePaymentStatus string

const (
	ChargePaymentStatusSuccess ChargePaymentStatus = "Success"
	ChargePaymentStatusPending ChargePaymentStatus = "Pending"
	ChargePaymentStatusError   ChargePaymentStatus = "Error"
	ChargeGatewayError         ChargePaymentStatus = "gateway_error"
)

type ChargePaymentResponse struct {
	Status        ChargePaymentStatus `json:"status"`
	Retryable     bool                `json:"retryable"`
	Psp           domain.Gateway      `json:"psp"`
	PspId         string              `json:"psp_id"`
	Reference     string              `json:"reference"`
	Currency      domain.Currency     `json:"currency"`
	AmountCharged int64               `json:"amount_charged"`
	PaymentType   string              `json:"payment_type"`
	ErrorReason   string              `json:"error_reason"`
	ErrorCode     string              `json:"error_code"`
	PspResponse   any                 `json:"psp_response"`
}

type InitPaymentResponse struct {
	PspResponse any
}

type PaymentWebhookType string

const (
	WebhookNoop             PaymentWebhookType = "noop"
	WebhookPaymentSuccess   PaymentWebhookType = "payment.success"
	WebhookPaymentRefunded  PaymentWebhookType = "payment.refunded"
	WebhookRecurringSuccess PaymentWebhookType = "recurring.success"
)

// PspPaymentMethod represents a payment method as known by the PSP.
type PspPaymentMethod struct {
	PspId       string `json:"psp_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	IsRecurring bool   `json:"is_recurring"`
	Token       string `json:"token"`
}

// PspPayment represents a payment as returned by the PSP.
type PspPayment struct {
	Currency    string               `json:"currency"`
	Reference   string               `json:"reference"`
	PspId       string               `json:"psp_id"`
	Amount      int64                `json:"amount"`
	Status      domain.PaymentStatus `json:"status"`
	PaidAt      time.Time            `json:"paid_at"`
	PspFee      int                  `json:"psp_fee"`
	PlatformFee int                  `json:"platform_fee"`
}

// PspCustomer represents a customer as known by the PSP.
type PspCustomer struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	PspId     string `json:"psp_id"`
}

// PaymentWebhookContext is the parsed context from an incoming payment webhook.
type PaymentWebhookContext struct {
	Type          PaymentWebhookType `json:"type"`
	OrgId         string             `json:"org_id"`
	OrderId       string             `json:"order_id"`
	Psp           domain.Gateway     `json:"psp"`
	Status        string             `json:"status"`
	Payment       PspPayment         `json:"payment"`
	Customer      PspCustomer        `json:"customer"`
	PaymentMethod PspPaymentMethod   `json:"payment_method"`
	RawData       []byte             `json:"raw_data"`
}

func ParsePaymentWebhookContext(data any) (PaymentWebhookContext, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return PaymentWebhookContext{}, errors.New("failed to marshal data to JSON")
	}

	var payload PaymentWebhookContext
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return PaymentWebhookContext{}, errors.New("failed to unmarshal JSON to PaymentWebhookContext")
	}
	return payload, nil
}

// GatewayAdapter is a port that payment adapter packages implement.
// The GatewayFactory uses this to create gateway/webhook parser instances
// without importing adapter packages directly.
type GatewayAdapter interface {
	CreateGateway(settingsJSON string) (domain.GatewayProvider, error)
	CreateWebhookParser() domain.WebhookParser
}

// PaymentWebhookPayload wraps a payment webhook for processing.
type PaymentWebhookPayload struct {
	Psp domain.Gateway `json:"psp"`
	// Signature is the PSP-supplied HMAC of the raw body, taken from
	// the request header that PSP uses (X-Paystack-Signature for
	// Paystack, Cko-Signature for Checkout.com, etc). Passing it
	// here means the parser doesn't need access to the HTTP request.
	Signature string `json:"signature"`
	Data      string `json:"data"`
}

// AuthnWebhookPayload wraps an authentication webhook for processing.
type AuthnWebhookPayload struct {
	Provider string `json:"provider"`
	Data     string `json:"data"`
}
