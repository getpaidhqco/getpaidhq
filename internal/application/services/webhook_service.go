package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/interfaces/webhooks"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"time"
)

type WebhookService struct {
	logger                 logger.Logger
	gatewayFactory         factories.GatewayFactory
	workflowEngine         interfaces.Engine
	idempotencyRepo        repositories.IdempotencyKeyRepository
	subscriptionRepository repositories.SubscriptionRepository
}

func NewWebhookService(
	logger logger.Logger,
	gatewayFactory factories.GatewayFactory,
	workflowEngine interfaces.Engine,
	idempotencyRepo repositories.IdempotencyKeyRepository,
	subscriptionRepository repositories.SubscriptionRepository,
) webhooks.WebhookService {
	return WebhookService{
		logger:                 logger,
		gatewayFactory:         gatewayFactory,
		workflowEngine:         workflowEngine,
		idempotencyRepo:        idempotencyRepo,
		subscriptionRepository: subscriptionRepository,
	}
}

// HandlePaymentWebhook parses a payment webhook and checks if it is valid. If valid, it publishes
// a payment event to the event bus.
func (s WebhookService) HandlePaymentWebhook(ctx context.Context, payload webhooks.PaymentWebhookPayload) error {
	s.logger.Infof("HandlePaymentWebhook: %s", payload.Data)

	hash := md5.Sum([]byte(payload.Data))
	hashHex := hex.EncodeToString(hash[:])
	// Check if the idempotency key already exists
	exists, err := s.idempotencyRepo.Exists(ctx, hashHex)
	if err != nil {
		s.logger.Errorf("failed to check idempotency key", err.Error())
		return events.NewQueueHandlerError("failed to check idempotency key", false, err)
	}
	if exists {
		s.logger.Info("Webhook already processed")
		return nil
	}

	parser := s.gatewayFactory.NewWebhookParser(payload.Psp)
	if parser == nil {
		s.logger.Error("failed to create webhook parser")
		return events.NewQueueHandlerError("failed to create webhook parser", false, err)
	}

	err = parser.ValidateWebhook(ctx, []byte(payload.Data))
	if err != nil {
		s.logger.Error("Failed to validate webhook", err.Error())
		return events.NewQueueHandlerError("Failed to validate webhook", false, err)
	}

	webhook, err := parser.ParseWebhook(ctx, []byte(payload.Data))
	if err != nil {
		s.logger.Error("failed to parse webhook", "err", err.Error())
		return events.NewQueueHandlerError("failed to parse webhook", false, err)
	}

	s.logger.Infof("Webhook parsed [%s][%s][%s][%s]", webhook.OrgId, webhook.OrderId, webhook.Psp, webhook.Type)

	switch webhook.Type {
	case payment_providers.PaymentSuccess:
		// start workflow
		s.workflowEngine.StartWorkflow(ctx, interfaces.PaymentSuccess, webhook)
	case payment_providers.RecurringSuccess:
		subs, err := s.subscriptionRepository.FindByOrderId(ctx, webhook.OrgId, webhook.OrderId)
		if err != nil {
			s.logger.Error("Failed to get subscriptions", err.Error())
			return err
		}
		if len(subs) == 0 {
			s.logger.Error("No subscriptions found for order")
			return nil
		}
		subscription := subs[0]

		chargeResult := payments.ChargeResult{
			Psp:         payload.Psp,
			Amount:      webhook.Payment.Amount,
			Status:      payments.PaymentStatusSucceeded,
			ErrorReason: "",
			ErrorCode:   "",
			Currency:    webhook.Payment.Currency,
			PspId:       webhook.Payment.PspId,
			Reference:   webhook.Payment.Reference,
			ProcessedAt: webhook.Payment.PaidAt,
			RawData:     string(webhook.RawData),
		}

		// signal the workflow
		err = s.workflowEngine.SignalSubscriptionWorkflow(ctx, "webhook-signal", subscription, chargeResult)

	case payment_providers.PaymentRefunded:
		// start workflow
		_, err := s.workflowEngine.StartWorkflow(ctx, interfaces.PaymentRefunded, webhook)
		if err != nil {
			s.logger.Errorf("Failed to start workflow %v", err.Error())
			return err
		}
	default:
		s.logger.Info("Unknown webhook type", "type", webhook.Type)

	}

	// Store the idempotency key
	err = s.idempotencyRepo.Create(ctx, hashHex, time.Now().Add(24*time.Hour))
	if err != nil {
		s.logger.Errorf("failed to store idempotency key", err.Error())
		return events.NewQueueHandlerError("failed to store idempotency key", false, err)
	}
	return nil
}
