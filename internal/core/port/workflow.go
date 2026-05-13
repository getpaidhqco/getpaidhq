package port

import (
	"context"
	"getpaidhq/internal/core/domain"
)

// Engine is the interface for the workflow orchestration engine.
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

type WorkflowResult struct {
	Success bool
	Message string
	Payload any
}

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

// PaymentSuccessWorkflow defines the interface for the payment success workflow.
type PaymentSuccessWorkflow interface {
	CompleteOrder(ctx context.Context, order domain.Order) (domain.Order, error)
}

// CompleteOrderStepInput is the input for the complete order workflow step.
type CompleteOrderStepInput struct {
	PaymentContext PaymentWebhookContext
}

// Workflow represents a runnable workflow.
type Workflow interface {
	Start(ctx any, payload any) (WorkflowResult, error)
}

// WorkflowSteps defines the steps that can be executed within a workflow.
type WorkflowSteps interface {
	CompleteOrder(ctx context.Context, data CompleteOrderStepInput) (WorkflowResult, error)
}

// WorkflowPayload wraps data and steps for workflow execution.
type WorkflowPayload struct {
	Data  any
	Steps WorkflowSteps
}

// PaymentSuccessWorkflowPayload is the payload for the payment success workflow.
type PaymentSuccessWorkflowPayload struct {
}
