package steps

import (
	"context"
	"getpaidhq/internal/core/port"
)

type OutgoingWebhookSteps struct {
	logger            port.Logger
	whService         port.WebhookSubscriptionService
	webhookRepository port.WebhookSubscriptionRepository
	settingRepository port.SettingRepository
	pubsub            port.PubSub
}

func NewOutgoingWebhookSteps(
	logger port.Logger,
	webhookRepository port.WebhookSubscriptionRepository,
	settingRepository port.SettingRepository,
	whService port.WebhookSubscriptionService,
	pubsub port.PubSub,
) *OutgoingWebhookSteps {
	return &OutgoingWebhookSteps{
		logger:            logger,
		whService:         whService,
		webhookRepository: webhookRepository,
		settingRepository: settingRepository,
		pubsub:            pubsub,
	}
}

func (s *OutgoingWebhookSteps) SendWebhook(ctx context.Context, data port.OutgoingWebhookPayload) error {
	s.logger.Info("SendWebhook")
	err := s.whService.SendWebhook(ctx, data)
	if err != nil {
		s.logger.Error("Failed to send webhook", "error", err.Error())
		return err
	}
	return nil
}
