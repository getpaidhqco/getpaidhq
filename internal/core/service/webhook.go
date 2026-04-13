package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"time"
)

type WebhookService struct {
	logger                 port.Logger
	gatewayFactory         *GatewayFactory
	workflowEngine         port.Engine
	idempotencyRepo        port.IdempotencyKeyRepository
	subscriptionRepository port.SubscriptionRepository
}

func NewWebhookService(
	logger port.Logger,
	gatewayFactory *GatewayFactory,
	workflowEngine port.Engine,
	idempotencyRepo port.IdempotencyKeyRepository,
	subscriptionRepository port.SubscriptionRepository,
) *WebhookService {
	return &WebhookService{
		logger:                 logger,
		gatewayFactory:         gatewayFactory,
		workflowEngine:         workflowEngine,
		idempotencyRepo:        idempotencyRepo,
		subscriptionRepository: subscriptionRepository,
	}
}

// HandlePaymentWebhook parses a payment webhook and checks if it is valid. If valid, it publishes
// a payment event to the event bus.
func (s *WebhookService) HandlePaymentWebhook(ctx context.Context, payload port.PaymentWebhookPayload) error {
	s.logger.Info("handling payment webhook", "data", payload.Data)

	hash := md5.Sum([]byte(payload.Data))
	hashHex := hex.EncodeToString(hash[:])
	// Check if the idempotency key already exists
	exists, err := s.idempotencyRepo.Exists(ctx, hashHex)
	if err != nil {
		s.logger.Error("failed to check idempotency key", "error", err)
		return port.NewQueueHandlerError("failed to check idempotency key", false, err)
	}
	if exists {
		s.logger.Info("webhook already processed")
		return nil
	}

	parser := s.gatewayFactory.NewWebhookParser(payload.Psp)
	if parser == nil {
		s.logger.Error("failed to create webhook parser")
		return port.NewQueueHandlerError("failed to create webhook parser", false, err)
	}

	err = parser.ValidateWebhook(ctx, []byte(payload.Data))
	if err != nil {
		s.logger.Error("failed to validate webhook", "error", err)
		return port.NewQueueHandlerError("Failed to validate webhook", false, err)
	}

	webhook, err := parser.ParseWebhook(ctx, []byte(payload.Data))
	if err != nil {
		s.logger.Error("failed to parse webhook", "error", err)
		return port.NewQueueHandlerError("failed to parse webhook", false, err)
	}

	s.logger.Info("webhook parsed", "orgId", webhook.OrgId, "orderId", webhook.OrderId, "psp", webhook.Psp, "type", webhook.Type)

	switch webhook.Type {
	case domain.PaymentSuccess:
		// start workflow
		s.workflowEngine.StartWorkflow(ctx, port.WorkflowPaymentSuccess, webhook)
	case domain.RecurringSuccess:
		subs, err := s.subscriptionRepository.FindByOrderId(ctx, webhook.OrgId, webhook.OrderId)
		if err != nil {
			s.logger.Error("failed to get subscriptions", "error", err)
			return err
		}
		if len(subs) == 0 {
			s.logger.Error("no subscriptions found for order")
			return nil
		}
		subscription := subs[0]

		chargeResult := domain.ChargeResult{
			Psp:         domain.Gateway(payload.Psp),
			Amount:      webhook.Payment.Amount,
			Status:      domain.PaymentStatusSucceeded,
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

	case domain.PaymentRefunded:
		// start workflow
		_, err := s.workflowEngine.StartWorkflow(ctx, port.WorkflowPaymentRefunded, webhook)
		if err != nil {
			s.logger.Error("failed to start workflow", "error", err)
			return err
		}
	default:
		s.logger.Info("unknown webhook type", "type", webhook.Type)

	}

	// Store the idempotency key
	err = s.idempotencyRepo.Create(ctx, hashHex, time.Now().Add(24*time.Hour))
	if err != nil {
		s.logger.Error("failed to store idempotency key", "error", err)
		return port.NewQueueHandlerError("failed to store idempotency key", false, err)
	}
	return nil
}

// HandleAuthnWebhook processes an authentication webhook and logs the provider and data.
func (s *WebhookService) HandleAuthnWebhook(ctx context.Context, payload port.AuthnWebhookPayload) error {
	s.logger.Info("handling authn webhook", "provider", payload.Provider, "data", payload.Data)

	// Add your business logic here, e.g., validating the payload or triggering workflows.

	return nil
}
