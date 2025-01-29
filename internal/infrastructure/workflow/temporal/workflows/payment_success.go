package workflows

import (
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"

	"go.temporal.io/sdk/workflow"
)

// PaymentSuccessWorkflow executes tasks for processing a successful payment
func PaymentSuccessWorkflow(ctx workflow.Context, payload workflow.WorkflowContext) (result string, err error) {
	// step 1, mark the order as paid
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 1000 * time.Second,
	}
	ctx1 := workflow.WithActivityOptions(ctx, ao)
	logger := workflow.GetLogger(ctx)

	err = workflow.ExecuteActivity(ctx1, activities.CompleteOrderActivity, payload).Get(ctx1, nil)
	if err != nil {
		logger.Error("Failed to create activityu", "Error", err)
		return "", err
	}

	logger.Info("Workflow completed.")
	return "COMPLETED", nil
}
