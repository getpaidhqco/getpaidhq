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

// HandlePaymentWebhook parses a payment webhook and checks if it is valid. If valid, it publishes
// a payment event to the event bus.
func (s *WebhookService) HandlePaymentWebhook(ctx context.Context, input []byte) error {
	s.logger.Info("Webhook ")

	webhook, err := s.payments.ParseWebhook(ctx, input)
	if err != nil {
		s.logger.Errorf("failed to parse webhook", err.Error())
		return err
	}

	s.logger.Info("Webhook parsed", "org_id", webhook.OrgId)

	return nil
}
