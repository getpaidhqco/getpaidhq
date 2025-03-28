package activities

import (
	"context"
	"go.temporal.io/sdk/activity"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/workflow"
)

type OutgoingWebhookActivities struct {
	whService         interfaces.WebhookSubscriptionService
	webhookRepository repositories.WebhookSubscriptionRepository
	settingRepository repositories.SettingRepository
	pubsub            events.PubSub
}

func NewOutgoingWebhookActivities(
	webhookRepository repositories.WebhookSubscriptionRepository,
	settingRepository repositories.SettingRepository,
	whService interfaces.WebhookSubscriptionService,
	pubsub events.PubSub,
) OutgoingWebhookActivities {
	return OutgoingWebhookActivities{
		whService:         whService,
		webhookRepository: webhookRepository,
		settingRepository: settingRepository,
		pubsub:            pubsub,
	}
}

func (a *OutgoingWebhookActivities) SendWebhook(ctx context.Context, data workflow.OutgoingWebhookPayload) error {
	logger := activity.GetLogger(ctx)
	logger.Info("SendWebhook")

	err := a.whService.SendWebhook(ctx, data)
	if err != nil {
		logger.Error("Failed to send webhook", "Error", err)
		return err
	}

	return nil
}
