package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/interfaces/webhooks"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"time"
)

type WebhookService struct {
	logger          logger.Logger
	gatewayFactory  factories.GatewayFactory
	workflowEngine  interfaces.Engine
	idempotencyRepo repositories.IdempotencyKeyRepository
}

func NewWebhookService(
	logger logger.Logger,
	gatewayFactory factories.GatewayFactory,
	workflowEngine interfaces.Engine,
	idempotencyRepo repositories.IdempotencyKeyRepository,
) webhooks.WebhookService {
	return WebhookService{
		logger:          logger,
		gatewayFactory:  gatewayFactory,
		workflowEngine:  workflowEngine,
		idempotencyRepo: idempotencyRepo,
	}
}

// HandlePaymentWebhook parses a payment webhook and checks if it is valid. If valid, it publishes
// a payment event to the event bus.
func (s WebhookService) HandlePaymentWebhook(ctx context.Context, payload webhooks.PaymentWebhookPayload) error {
	s.logger.Infof("HandlePaymentWebhook: %s", string(payload.Data))

	hash := md5.Sum([]byte(payload.Data))
	hashHex := hex.EncodeToString(hash[:])
	// Check if the idempotency key already exists
	exists, err := s.idempotencyRepo.Exists(ctx, hashHex)
	if err != nil {
		s.logger.Errorf("failed to check idempotency key", err.Error())
		return err
	}
	if exists {
		s.logger.Info("Webhook already processed")
		return nil
	}

	// Store the idempotency key
	err = s.idempotencyRepo.Create(ctx, hashHex, time.Now().Add(24*time.Hour))
	if err != nil {
		s.logger.Errorf("failed to store idempotency key", err.Error())
		return err
	}

	parser := s.gatewayFactory.NewWebhookParser(payload.Psp)
	err = parser.ValidateWebhook(ctx, []byte(payload.Data))
	if err != nil {
		s.logger.Error("Failed to validate webhook", err.Error())
		return err
	}

	webhook, err := parser.ParseWebhook(ctx, []byte(payload.Data))
	if err != nil {
		s.logger.Errorf("failed to parse webhook", err.Error())
		return err
	}

	s.logger.Info("Webhook parsed", "org_id", webhook.OrgId)

	s.startWorkflow(ctx, webhook)
	return nil
}

// TODO this needs to be handled by a worker from a queue
func (s WebhookService) startWorkflow(ctx context.Context, event payment_providers.PaymentWebhookContext) {
	switch event.Type {
	case payment_providers.PaymentSuccess:
		// start workflow
		s.workflowEngine.StartWorkflow(ctx, interfaces.PaymentSuccess, event)
	default:
		s.logger.Info("Unknown webhook type", "type", event.Type)

	}
}
