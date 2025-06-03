package webhooks

import (
	"context"
	"payloop/internal/domain/common"
)

type PaymentWebhookPayload struct {
	Psp  common.Gateway `json:"psp"`
	Data string         `json:"data"`
}
type AuthnWebhookPayload struct {
	Provider string `json:"provider"`
	Data     string `json:"data"`
}

type WebhookService interface {
	HandlePaymentWebhook(ctx context.Context, payload PaymentWebhookPayload) error
	HandleAuthnWebhook(ctx context.Context, payload AuthnWebhookPayload) error
}
