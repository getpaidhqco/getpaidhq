package workflows

import (
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/domain/workflow"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"

	temporal "go.temporal.io/sdk/workflow"
)

// OutgoingWebhookWorkflow is a Temporal Workflow that delivers a webhook payload to a subscriber
func OutgoingWebhookWorkflow(ctx temporal.Context, payload workflow.OutgoingWebhookPayload) (workflow.Result, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("OutgoingWebhookWorkflow started")

	var a *activities.OutgoingWebhookActivities

	// ACTIVITY
	// Send the webhook
	var sendWebhookResult workflow.Result
	ctx1 := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 15 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			MaximumAttempts:    5,
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
		},
	})
	err := temporal.ExecuteActivity(ctx1, a.SendWebhook, payload).
		Get(ctx1, &sendWebhookResult)
	if err != nil {
		logger.Error("[SendWebhook] failed with error: ", "Error", err.Error())
		return workflow.Result{}, temporalio.NewNonRetryableApplicationError("SendWebhook failed", "webhook", err)
	}

	logger.Info("Workflow completed.")
	return sendWebhookResult, nil
}
