package workflows

import (
	"payloop/internal/domain/workflow"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"

	temporal "go.temporal.io/sdk/workflow"
)

// Execute executes tasks for processing a successful payment
func PaymentSuccessWorkflow(ctx temporal.Context, payload interface{}) (workflow.Result, error) {
	logger := temporal.GetLogger(ctx)
	// step 1, mark the order as paid
	ao := temporal.ActivityOptions{
		StartToCloseTimeout: 1000 * time.Second,
	}
	ctx1 := temporal.WithActivityOptions(ctx, ao)

	// Complete Order
	err := temporal.ExecuteActivity(ctx1, activities.CompleteOrder, workflow.CompleteOrderStepInput{
		OrgId:   "mollie",
		OrderId: "",
	}).Get(ctx1, nil)

	if err != nil {
		logger.Error("Failed to create activityu", "Error", err)
		return workflow.Result{}, nil
	}

	logger.Info("Workflow completed.")
	return workflow.Result{}, nil
}
