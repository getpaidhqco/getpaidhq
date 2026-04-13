package workflows

import (
	"errors"
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/adapter/temporal/activities"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"time"

	temporal "go.temporal.io/sdk/workflow"
)

// Execute executes tasks for processing a payment refunded event
func PaymentRefunded(ctx temporal.Context, payload domain.PaymentWebhookContext) (port.WorkflowResult, error) {
	logger := temporal.GetLogger(ctx)

	// parse the data to make sure we have what we need
	paymentWebhookContext, err := domain.ParsePaymentWebhookContext(payload)
	if err != nil {
		logger.Error("invalid payload data", "error", err)
		return port.WorkflowResult{}, errors.New("invalid payload data, expected domain.PaymentWebhookContext ")
	}

	var a *activities.OrderActivities

	// ACTIVITY
	// Complete the Order
	var completeOrderResult port.WorkflowResult
	ctx1 := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
		},
	})
	err = temporal.ExecuteActivity(ctx1, a.HandlePaymentRefundedEvent, paymentWebhookContext).
		Get(ctx1, &completeOrderResult)
	if err != nil {
		logger.Error("HandlePaymentRefundedEvent failed", "error", err)
		return port.WorkflowResult{}, temporalio.NewApplicationError("handle refund event failed", "", err)
	}

	logger.Info("[PaymentRefunded] Workflow completed.")
	return port.WorkflowResult{
		Success: true,
		Message: "PaymentRefunded completed",
		Payload: completeOrderResult.Payload,
	}, nil
}
