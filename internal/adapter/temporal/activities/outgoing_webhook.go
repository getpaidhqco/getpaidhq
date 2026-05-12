package activities

import (
	"context"
	"go.temporal.io/sdk/activity"
	"getpaidhq/internal/core/port"
)

type OutgoingWebhookActivities struct {
	whService         port.WebhookSubscriptionService
	webhookRepository port.WebhookSubscriptionRepository
	settingRepository port.SettingRepository
	pubsub            port.PubSub
}

func NewOutgoingWebhookActivities(
	webhookRepository port.WebhookSubscriptionRepository,
	settingRepository port.SettingRepository,
	whService port.WebhookSubscriptionService,
	pubsub port.PubSub,
) OutgoingWebhookActivities {
	return OutgoingWebhookActivities{
		whService:         whService,
		webhookRepository: webhookRepository,
		settingRepository: settingRepository,
		pubsub:            pubsub,
	}
}

func (a *OutgoingWebhookActivities) SendWebhook(ctx context.Context, data port.OutgoingWebhookPayload) error {
	logger := activity.GetLogger(ctx)
	logger.Info("SendWebhook")

	err := a.whService.SendWebhook(ctx, data)
	if err != nil {
		logger.Error("Failed to send webhook", "Error", err)
		return err
	}

	return nil
}
