package workflow

import (
	"context"
	"payloop/internal/domain/payment_providers"
)

type Engine interface {
	StartWorkflow(ctx context.Context, id WorkflowType, payload interface{}) (Result, error)
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
	PaymentSuccess WorkflowType = "payment.success"
)
