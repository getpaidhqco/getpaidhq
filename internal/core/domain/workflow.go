package domain

import (
	"context"
	pubsub "payloop/internal/application/lib/events"
)

type OutgoingWebhookPayload struct {
	WebhookSubscription WebhookSubscription
	Event               pubsub.Payload
}

type PaymentRefundedPayload struct {
	refund Refund
}

type PaymentSuccessWorkflow interface {
	CompleteOrder(ctx context.Context, order Order) (Order, error)
}
