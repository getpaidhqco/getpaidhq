package activities

import (
	"payloop/internal/domain/workflow"
	"payloop/internal/lib"
)

type CompleteOrderActivity struct {
	logger lib.Logger
}

func NewCompleteOrderActivity(logger lib.Logger) workflow.Step {
	return CompleteOrderActivity{logger: logger}
}

func (s CompleteOrderActivity) Execute(ctx interface{}, payload interface{}) (workflow.Result, error) {
	s.logger.Info("CompleteOrderActivity.", "payload", payload)

	return workflow.Result{}, nil
}
