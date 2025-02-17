package workflows

import (
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/workflow"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"

	temporal "go.temporal.io/sdk/workflow"
)

// OutgoingWebhookWorkflow is a Temporal Workflow that delivers a webhook payload to a subscriber
func OutgoingWebhookWorkflow(ctx temporal.Context, payload workflow.OutgoingWebhookPayload) (interfaces.Result, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("OutgoingWebhookWorkflow started")

	var a *activities.OutgoingWebhookActivities

	// ACTIVITY
	// Send the webhook
	ctx1 := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 15 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
		},
	})
	err := temporal.ExecuteActivity(ctx1, a.SendWebhook, payload).
		Get(ctx1, nil)
	if err != nil {
		logger.Error("[SendWebhook] failed with error: ", "Error", err.Error())
		// Todo the webhook delivery failed and wont be retried.
		// We should log the error and return a non-retryable error
		return interfaces.Result{}, temporalio.NewNonRetryableApplicationError("SendWebhook failed", "webhook", err)
	}

	logger.Info("[outgoing_webhook] Workflow completed.")
	return interfaces.Result{
		Success: true,
		Message: "sent",
		Payload: nil,
	}, nil
}
