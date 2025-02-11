package activities

import (
	"context"
	"go.temporal.io/sdk/activity"
	"payloop/internal/application/lib/events"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/workflow"
)

type OutgoingWebhookActivities struct {
	webhookRepository repositories.WebhookSubscriptionRepository
	settingRepository repositories.SettingRepository
	pubsub            events.PubSub
}

func NewOutgoingWebhookActivities(
	webhookRepository repositories.WebhookSubscriptionRepository,
	settingRepository repositories.SettingRepository,
	pubsub events.PubSub,
) OutgoingWebhookActivities {
	return OutgoingWebhookActivities{
		webhookRepository: webhookRepository,
		settingRepository: settingRepository,
		pubsub:            pubsub,
	}
}

func (a *OutgoingWebhookActivities) SendWebhook(ctx context.Context, data workflow.OutgoingWebhookPayload) (workflow.Result, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("SendWebhook")

	return workflow.Result{
		Success: true,
		Message: "webhook sent",
		Payload: data,
	}, nil
}
