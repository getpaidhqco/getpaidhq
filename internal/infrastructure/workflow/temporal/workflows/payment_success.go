package workflows

import (
	"errors"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/workflow"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"

	temporal "go.temporal.io/sdk/workflow"
)

// Execute executes tasks for processing a successful payment
func PaymentSuccessWorkflow(ctx temporal.Context, payload workflow.WorkflowPayload) (workflow.Result, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("PaymentSuccessWorkflow started")

	// parse the data to make sure we have what we need
	wfData, err := payment_providers.ParsePaymentWebhookContext(payload.Data)
	if err != nil {
		logger.Error("Invalid payload data", "err", err.Error())
		return workflow.Result{}, errors.New("invalid payload data, expected payment_providers.PaymentWebhookContext ")
	}

	// step 1, mark the order as paid
	ao := temporal.ActivityOptions{
		StartToCloseTimeout: 1000 * time.Second,
	}
	ctx1 := temporal.WithActivityOptions(ctx, ao)

	var a *activities.OrderActivities

	// Complete Order
	var result workflow.Result
	err = temporal.ExecuteActivity(ctx1, a.CompleteOrder, workflow.CompleteOrderStepInput{
		PaymentContext: wfData,
	}).Get(ctx1, &result)

	if err != nil {
		logger.Error("a.CompleteOrder", "Error", err)
		return workflow.Result{}, nil
	}

	logger.Info("Workflow completed.")
	return workflow.Result{
		Success: true,
		Message: "PaymentSuccessWorkflow completed",
		Payload: result.Payload,
	}, nil
}
