package interfaces

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/payment_providers"
)

type WorkflowService interface {
	HandleOutboundWebhook(topic string, data []byte)
}

type Engine interface {
	StartWorkflow(ctx context.Context, id WorkflowType, payload interface{}) (Result, error)
	StartSubscriptionWorkflow(ctx context.Context, subscription entities.Subscription) (Result, error)
}

type Workflow interface {
	Start(ctx interface{}, payload interface{}) (Result, error)
}
type Steps interface {
	CompleteOrder(ctx context.Context, data CompleteOrderStepInput) (Result, error)
}

type PaymentSuccessWorkflowPayload struct {
}

type CompleteOrderStepInput struct {
	PaymentContext payment_providers.PaymentWebhookContext
}

type Result struct {
	Success bool
	Message string
	Payload interface{}
}

type WorkflowPayload struct {
	Data  interface{}
	Steps Steps
}

type WorkflowType string

const (
	PaymentSuccess  WorkflowType = "payment.success"
	OutgoingWebhook WorkflowType = "webhook"
	Subscription    WorkflowType = "subscription"
)
