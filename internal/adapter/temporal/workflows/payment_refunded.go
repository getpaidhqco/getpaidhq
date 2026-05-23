package workflows

import (
	"time"

	temporalio "go.temporal.io/sdk/temporal"
	temporal "go.temporal.io/sdk/workflow"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// PaymentRefunded handles a refund event. Mirrors
// internal/adapter/hatchet/workflows/payment_refunded.go.
func PaymentRefunded(ctx temporal.Context, input domain.PaymentWebhookContext) (port.WorkflowResult, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("PaymentRefunded started", "orgId", input.OrgId, "orderId", input.OrderId)

	var act *activities.OrderActivities

	actCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
			MaximumAttempts:    10,
		},
	})
	var payment domain.Payment
	if err := temporal.ExecuteActivity(actCtx, act.HandlePaymentRefundedEvent, input).
		Get(actCtx, &payment); err != nil {
		return port.WorkflowResult{}, err
	}
	return port.WorkflowResult{Success: true, Message: "Refund event processed", Payload: payment}, nil
}
