package services

import (
	"context"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/lib"
)

type WebhookService struct {
	logger   lib.Logger
	payments payment_providers.Gateway
}

func NewWebhookService(
	logger lib.Logger,
	payments payment_providers.Gateway,
) WebhookService {
	return WebhookService{
		logger:   logger,
		payments: payments,
	}
}

func (s *WebhookService) HandlePaymentWebhook(ctx context.Context, input []byte) error {
	s.logger.Info("Webhook ")

	s.payments.ParseWebhook(ctx, input)
	return nil
}
