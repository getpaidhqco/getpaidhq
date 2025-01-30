package workflow

import "context"

type Engine interface {
	StartWorkflow(ctx context.Context, id WorkflowType, payload interface{}) (Result, error)
}

type Workflow interface {
	Start(ctx interface{}, payload interface{}) (Result, error)
}
type PaymentSteps interface {
	CompleteOrder(payload CompleteOrderStepInput) (Result, error)
}

type PaymentSuccessWorkflowPayload struct {
}

type CompleteOrderStepInput struct {
	OrgId   string
	OrderId string
}

type Result struct {
	Success bool
	Message string
	Payload interface{}
}

type WorkflowPayload struct {
	Engine Engine
	Data   interface{}
}

type WorkflowType string

const (
	PaymentSuccess WorkflowType = "payment.success"
)
