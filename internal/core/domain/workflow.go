package domain

import "context"

// PubSubPayload represents a published event payload.
type PubSubPayload struct {
	Id        string `json:"id"`
	OrgId     string `json:"org_id"`
	Topic     string `json:"topic"`
	Data      any    `json:"data"`
	CreatedAt any    `json:"created_at"`
}

type OutgoingWebhookPayload struct {
	WebhookSubscription WebhookSubscription
	Event               PubSubPayload
}

type PaymentRefundedPayload struct {
	Refund Refund
}

type PaymentSuccessWorkflow interface {
	CompleteOrder(ctx context.Context, order Order) (Order, error)
}
