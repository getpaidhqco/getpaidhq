package workflows

import (
	"time"

	temporalio "go.temporal.io/sdk/temporal"
	temporal "go.temporal.io/sdk/workflow"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/port"
)

// OutgoingWebhookWorkflow delivers a single outbound webhook with retry.
// Mirrors internal/adapter/hatchet/workflows/outgoing_webhook.go.
func OutgoingWebhookWorkflow(ctx temporal.Context, payload port.OutgoingWebhookPayload) (port.WorkflowResult, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("OutgoingWebhookWorkflow started")

	var act *activities.OutgoingWebhookActivities

	actCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 15 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
		},
	})
	if err := temporal.ExecuteActivity(actCtx, act.SendWebhook, payload).
		Get(actCtx, nil); err != nil {
		return port.WorkflowResult{}, temporalio.NewNonRetryableApplicationError("SendWebhook failed", "webhook", err)
	}
	return port.WorkflowResult{Success: true, Message: "sent"}, nil
}
