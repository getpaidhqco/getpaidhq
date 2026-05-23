package port

import (
	"context"
	"getpaidhq/internal/core/domain"
)

// Engine is the workflow orchestration surface used by services.
//
// Hatchet and Temporal each provide a concrete implementation. Engine carries
// the subscription lifecycle methods (per-aggregate durable runner) plus a
// generic one-shot StartWorkflow for fire-and-forget DAG-style flows. The
// dunning surface lives on port.DunningEngine to keep this interface small;
// both adapters' concrete types satisfy both interfaces.
type Engine interface {
	StartWorkflow(ctx context.Context, id WorkflowType, payload any) (WorkflowResult, error)
	StartSubscriptionWorkflow(ctx context.Context, subscription domain.Subscription) error
	UpdateSubscriptionWorkflow(ctx context.Context, updateName string, subscription domain.Subscription) error
	CancelSubscriptionWorkflow(ctx context.Context, subscription domain.Subscription) error
	SignalSubscriptionWorkflow(ctx context.Context, signal string, subscription domain.Subscription, payload any) error
}

// WorkflowService handles outbound workflow operations (e.g., webhook delivery).
type WorkflowService interface {
	HandleOutboundWebhook(topic string, data []byte)
}

// WorkflowResult is the engine-agnostic return shape from a one-shot workflow.
type WorkflowResult struct {
	Success bool
	Message string
	Payload any
}

// WorkflowType identifies a one-shot workflow registered with the engine.
type WorkflowType string

const (
	WorkflowPaymentRefunded WorkflowType = "payment.refunded"
	WorkflowPaymentSuccess  WorkflowType = "payment.success"
	WorkflowOutgoingWebhook WorkflowType = "webhook"
	WorkflowSubscription    WorkflowType = "subscription"
)

// OutgoingWebhookPayload is the payload for sending outbound webhooks.
type OutgoingWebhookPayload struct {
	WebhookSubscription domain.WebhookSubscription
	Event               PubSubPayload
}

// PaymentRefundedPayload is the payload for the payment refunded workflow.
type PaymentRefundedPayload struct {
	Refund domain.Refund
}
