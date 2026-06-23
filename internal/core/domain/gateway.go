package domain

import (
	"context"
	"time"
)

// WebhookParser verifies and parses an incoming PSP webhook.
//
// ValidateWebhook MUST cryptographically verify `signature` against the
// raw `data` using a constant-time comparison. A nil return means the
// payload is authentic; any non-nil return means we reject the event,
// regardless of how good the body looks.
type WebhookParser interface {
	ValidateWebhook(ctx context.Context, data []byte, signature string) error
	ParseWebhook(ctx context.Context, data []byte) (PaymentWebhookContext, error)
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
	Currency    string        `json:"currency"`
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
