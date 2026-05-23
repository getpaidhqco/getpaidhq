package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
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
	s.logger.Infof("HandlePaymentWebhook: %s", payload.Data)

	hash := md5.Sum([]byte(payload.Data))
	hashHex := hex.EncodeToString(hash[:])
	// Check if the idempotency key already exists
	exists, err := s.idempotencyRepo.Exists(ctx, hashHex)
	if err != nil {
		s.logger.Errorf("failed to check idempotency key: %s", err.Error())
		return lib.NewCustomError(lib.InternalError, "failed to check idempotency key", err)
	}
	if exists {
		s.logger.Info("Webhook already processed")
		return nil
	}

	parser := s.gatewayFactory.NewWebhookParser(payload.Psp)
	if parser == nil {
		s.logger.Error("failed to create webhook parser")
		return lib.NewCustomError(lib.InternalError, "failed to create webhook parser", err)
	}

	err = parser.ValidateWebhook(ctx, []byte(payload.Data))
	if err != nil {
		s.logger.Error("Failed to validate webhook", err.Error())
		return lib.NewCustomError(lib.InternalError, "Failed to validate webhook", err)
	}

	webhook, err := parser.ParseWebhook(ctx, []byte(payload.Data))
	if err != nil {
		s.logger.Error("failed to parse webhook", "err", err.Error())
		return lib.NewCustomError(lib.InternalError, "failed to parse webhook", err)
	}

	s.logger.Infof("Webhook parsed [%s][%s][%s][%s]", webhook.OrgId, webhook.OrderId, webhook.Psp, webhook.Type)

	switch webhook.Type {
	case domain.PaymentSuccess:
		// start workflow
		s.workflowEngine.StartWorkflow(ctx, port.WorkflowPaymentSuccess, webhook)
	case domain.RecurringSuccess:
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
			s.logger.Errorf("Failed to start workflow %v", err.Error())
			return err
		}
	default:
		s.logger.Info("Unknown webhook type", "type", webhook.Type)

	}

	// Store the idempotency key
	err = s.idempotencyRepo.Create(ctx, hashHex, time.Now().Add(24*time.Hour))
	if err != nil {
		s.logger.Errorf("failed to store idempotency key: %s", err.Error())
		return lib.NewCustomError(lib.InternalError, "failed to store idempotency key", err)
	}
	return nil
}

// HandleAuthnWebhook processes an authentication webhook and logs the provider and data.
func (s *WebhookService) HandleAuthnWebhook(ctx context.Context, payload port.AuthnWebhookPayload) error {
	s.logger.Infof("HandleAuthnWebhook: Provider=%s, Data=%s", payload.Provider, payload.Data)

	// Add your business logic here, e.g., validating the payload or triggering workflows.

	return nil
}
