package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"time"
)

type WebhookService struct {
	logger          logger.Logger
	payments        payment_providers.Gateway
	workflowEngine  interfaces.Engine
	idempotencyRepo repositories.IdempotencyKeyRepository
}

func NewWebhookService(
	logger logger.Logger,
	payments payment_providers.Gateway,
	workflowEngine interfaces.Engine,
	idempotencyRepo repositories.IdempotencyKeyRepository,
) WebhookService {
	return WebhookService{
		logger:          logger,
		payments:        payments,
		workflowEngine:  workflowEngine,
		idempotencyRepo: idempotencyRepo,
	}
}

// HandlePaymentWebhook parses a payment webhook and checks if it is valid. If valid, it publishes
// a payment event to the event bus.
func (s *WebhookService) HandlePaymentWebhook(ctx context.Context, input []byte) error {
	s.logger.Info("Webhook ")

	hash := md5.Sum(input)
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
		s.workflowEngine.StartWorkflow(ctx, interfaces.PaymentSuccess, event)
	default:
		s.logger.Info("Unknown webhook type", "type", event.Type)

	}
}
