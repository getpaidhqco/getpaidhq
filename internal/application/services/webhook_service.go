package services

import (
	"context"
	"payloop/internal/lib"
)

type WebhookService struct {
	logger lib.Logger
}

func NewWebhookService(
	logger lib.Logger,
) WebhookService {
	return WebhookService{
		logger: logger,
	}
}

func (s *WebhookService) HandlePaymentWebhook(ctx context.Context, input []byte) error {
	s.logger.Info("Webhook ")
	return nil
}
