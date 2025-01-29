package workflows

import (
	"payloop/internal/domain/workflow"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/lib"
	"time"

	temporal "go.temporal.io/sdk/workflow"
)

type PaymentSuccessWorkflow struct {
	completeOrderStep workflow.Step
	logger            lib.Logger
}

func NewPaymentSuccessWorkflow(logger lib.Logger) PaymentSuccessWorkflow {
	return PaymentSuccessWorkflow{
		logger: logger,
	}
}

// Execute executes tasks for processing a successful payment
func (p PaymentSuccessWorkflow) Start(ctx interface{}, payload interface{}) (workflow.Result, error) {
	temporalCtx := ctx.(temporal.Context)

	// step 1, mark the order as paid
	ao := temporal.ActivityOptions{
		StartToCloseTimeout: 1000 * time.Second,
	}
	ctx1 := temporal.WithActivityOptions(temporalCtx, ao)
	logger := temporal.GetLogger(temporalCtx)

	var a *activities.CompleteOrderActivity
	err := temporal.ExecuteActivity(ctx1, a.Execute, payload).Get(ctx1, nil)
	if err != nil {
		logger.Error("Failed to create activityu", "Error", err)
		return workflow.Result{}, nil
	}

	logger.Info("Workflow completed.")
	return workflow.Result{}, nil
}
