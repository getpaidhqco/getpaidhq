package services

import (
	"context"
	"go.temporal.io/sdk/workflow"
	"payloop/internal/api/controllers"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/infrastructure/workflow/temporal/activities"
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

// TODO this needs to be handled by a worker from a queue
func startWorkflow(event payment_providers.PaymentWebhookContext) {
	switch event.Type {
	case payment_providers.PaymentSuccess:
		// start workflow
		workflow.ExecuteActivity(ctx1, activities.CompleteOrderActivity, payload).Get(ctx1, nil)

	}
}
