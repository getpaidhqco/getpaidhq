package workflows

import (
	"errors"
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"

	temporal "go.temporal.io/sdk/workflow"
)

// Execute executes tasks for processing a payment refunded event
func PaymentRefunded(ctx temporal.Context, payload payment_providers.PaymentWebhookContext) (interfaces.Result, error) {
	logger := temporal.GetLogger(ctx)

	// parse the data to make sure we have what we need
	paymentWebhookContext, err := payment_providers.ParsePaymentWebhookContext(payload)
	if err != nil {
		logger.Error("Invalid payload data", "err", err.Error())
		return interfaces.Result{}, errors.New("invalid payload data, expected payment_providers.PaymentWebhookContext ")
	}

	var a *activities.OrderActivities

	// ACTIVITY
	// Complete the Order
	var completeOrderResult interfaces.Result
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
		logger.Error("[HandlePaymentRefundedEvent] failed with error: ", "Error", err.Error())
		return interfaces.Result{}, temporalio.NewApplicationError("handle refund event failed", "", err)
	}

	logger.Info("[PaymentRefunded] Workflow completed.")
	return interfaces.Result{
		Success: true,
		Message: "PaymentRefunded completed",
		Payload: completeOrderResult.Payload,
	}, nil
}
