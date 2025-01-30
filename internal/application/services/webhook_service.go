package services

import (
	"context"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/workflow"
	"payloop/internal/lib"
)

type WebhookService struct {
	logger         lib.Logger
	payments       payment_providers.Gateway
	workflowEngine workflow.Engine
}

func NewWebhookService(
	logger lib.Logger,
	payments payment_providers.Gateway,
	workflowEngine workflow.Engine,
) WebhookService {
	return WebhookService{
		logger:         logger,
		payments:       payments,
		workflowEngine: workflowEngine,
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

	// TODO Instead place this event in a queue and let a worker handle it
	s.startWorkflow(ctx, webhook)
	return nil
}

// TODO this needs to be handled by a worker from a queue
func (s *WebhookService) startWorkflow(ctx context.Context, event payment_providers.PaymentWebhookContext) {
	switch event.Type {
	case payment_providers.PaymentSuccess:
		// start workflow
		s.workflowEngine.StartWorkflow(ctx, workflow.PaymentSuccess, event)
	default:
		s.logger.Info("Unknown webhook type", "type", event.Type)

	}
}
