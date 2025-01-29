package workflow

import "context"

type Engine interface {
	StartWorkflow(ctx context.Context, id string, payload interface{}) (Result, error)
}

type Workflow interface {
	Start(ctx interface{}, payload interface{}) (Result, error)
}
type PaymentSteps interface {
	CompleteOrder(payload CompleteOrderStepInput) (Result, error)
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

type WorkflowContext struct {
	EventId string
	OrderId string
}
