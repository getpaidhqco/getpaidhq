package activities

import (
	"context"

	"go.temporal.io/sdk/activity"

	"getpaidhq/internal/core/port"
)

// OutgoingWebhookActivities is the Temporal mirror of
// internal/adapter/hatchet/steps/outgoing_webhook_steps.go. One activity that
// delegates to the engine-agnostic WebhookSubscriptionService.
type OutgoingWebhookActivities struct {
	whService port.WebhookSubscriptionService
}

func NewOutgoingWebhookActivities(whService port.WebhookSubscriptionService) OutgoingWebhookActivities {
	return OutgoingWebhookActivities{whService: whService}
}

func (a *OutgoingWebhookActivities) SendWebhook(ctx context.Context, data port.OutgoingWebhookPayload) error {
	logger := activity.GetLogger(ctx)
	logger.Info("SendWebhook")

	if err := a.whService.SendWebhook(ctx, data); err != nil {
		logger.Error("Failed to send webhook", "Error", err)
		return err
	}
	return nil
}
