package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib/errors"
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

// HandlePaymentWebhook validates, parses, and dispatches a PSP webhook.
//
// Idempotency model (CHANGED — read carefully):
//
//	The idempotency key is CLAIMED before any side effect. If we lose the
//	race to a concurrent retry, we short-circuit and the caller sees a
//	clean 200 — exactly what an "already processed" delivery should
//	produce. If we DO win the claim but then any downstream step fails,
//	we RELEASE the key (best-effort delete) so the PSP's retry can run
//	the work — otherwise a transient downstream failure would silently
//	drop the event.
//
// Hash:
//
//	SHA-256 of "<psp>:<raw-body>". Including the PSP namespaces by
//	provider so two PSPs sending identical JSON shapes can't collide.
//	(MD5 was previously used here; collisions are cheap to construct.)
func (s *WebhookService) HandlePaymentWebhook(ctx context.Context, payload port.PaymentWebhookPayload) error {
	// Don't log the raw payload — webhooks carry customer PII and PSP-
	// side payment metadata that would put log aggregation in PCI scope.
	s.logger.Info("HandlePaymentWebhook", "psp", payload.Psp, "size", len(payload.Data))

	hash := sha256.Sum256([]byte(string(payload.Psp) + ":" + payload.Data))
	hashHex := hex.EncodeToString(hash[:])

	// Claim FIRST. ON CONFLICT DO NOTHING in the repo turns the
	// concurrent-retry case into a clean "claimed=false" rather than a
	// 500 to the PSP.
	claimed, err := s.idempotencyRepo.Claim(ctx, hashHex, time.Now().Add(24*time.Hour))
	if err != nil {
		s.logger.Error("idempotency claim failed", "err", err.Error())
		return errors.NewCustomError(errors.InternalError, "idempotency claim failed", err)
	}
	if !claimed {
		// Already processed by a sibling delivery; nothing to do.
		s.logger.Info("webhook already processed (idempotency hit)")
		return nil
	}

	// From here on, on any error we RELEASE the claim so the PSP's retry
	// can run the work. Without this, a flaky downstream would drop the
	// event silently after we'd already taken the key.
	release := func(reason string, cause error) error {
		if relErr := s.idempotencyRepo.Release(ctx, hashHex); relErr != nil {
			s.logger.Error("idempotency release failed (event will be re-tried by PSP only if our row TTLs out)",
				"key", hashHex, "err", relErr.Error())
		}
		return errors.NewCustomError(errors.InternalError, reason, cause)
	}

	parser := s.gatewayFactory.NewWebhookParser(payload.Psp)
	if parser == nil {
		s.logger.Error("no webhook parser registered", "psp", payload.Psp)
		return release("no webhook parser registered for psp", nil)
	}

	if err := parser.ValidateWebhook(ctx, []byte(payload.Data), payload.Signature); err != nil {
		// Signature failures are NOT release-worthy — a forged or
		// mis-signed delivery should not be retried by us; the PSP will
		// stop retrying once it sees the 4xx we surface upstream. But we
		// also can't tell here whether the parser returns "signature bad"
		// vs "transient infra error", so be conservative: release so the
		// PSP's retry gets through if the cause was transient. A forged
		// payload would just fail validation again and the PSP would
		// stop on its own backoff.
		s.logger.Error("webhook validation failed", "err", err.Error())
		return release("webhook validation failed", err)
	}

	webhook, err := parser.ParseWebhook(ctx, []byte(payload.Data))
	if err != nil {
		s.logger.Error("webhook parse failed", "err", err.Error())
		return release("webhook parse failed", err)
	}

	s.logger.Info("webhook parsed",
		"orgId", webhook.OrgId, "orderId", webhook.OrderId,
		"psp", webhook.Psp, "type", webhook.Type)

	switch webhook.Type {
	case domain.PaymentSuccess:
		// Engine errors propagate — previously this call's return values
		// were discarded, so engine outages would silently drop the
		// payment-success event. The PSP's retry is our safety net.
		if _, err := s.workflowEngine.StartWorkflow(ctx, port.WorkflowPaymentSuccess, webhook); err != nil {
			s.logger.Error("StartWorkflow(PaymentSuccess) failed", "err", err.Error())
			return release("failed to start payment-success workflow", err)
		}
	case domain.RecurringSuccess:
		subs, err := s.subscriptionRepository.FindByOrderId(ctx, webhook.OrgId, webhook.OrderId)
		if err != nil {
			s.logger.Error("FindByOrderId failed", "err", err.Error())
			return release("failed to load subscription for recurring charge", err)
		}
		if len(subs) == 0 {
			// No subscription means no signal to deliver. This is not an
			// error condition — the order may have already terminated.
			// Keep the claim so PSP retries don't reprocess.
			s.logger.Warn("no subscriptions found for recurring charge", "orgId", webhook.OrgId, "orderId", webhook.OrderId)
			return nil
		}
		subscription := subs[0]

		chargeResult := domain.ChargeResult{
			Psp:         domain.Gateway(payload.Psp),
			Amount:      webhook.Payment.Amount,
			Status:      domain.PaymentStatusSucceeded,
			Currency:    webhook.Payment.Currency,
			PspId:       webhook.Payment.PspId,
			Reference:   webhook.Payment.Reference,
			ProcessedAt: webhook.Payment.PaidAt,
			RawData:     string(webhook.RawData),
		}

		if err := s.workflowEngine.SignalSubscriptionWorkflow(ctx, "webhook-signal", subscription, chargeResult); err != nil {
			s.logger.Error("SignalSubscriptionWorkflow failed", "err", err.Error())
			return release("failed to signal subscription workflow", err)
		}

	case domain.PaymentRefunded:
		if _, err := s.workflowEngine.StartWorkflow(ctx, port.WorkflowPaymentRefunded, webhook); err != nil {
			s.logger.Error("StartWorkflow(PaymentRefunded) failed", "err", err.Error())
			return release("failed to start payment-refunded workflow", err)
		}
	default:
		// Unknown types are explicitly idempotency-claimed so a noisy PSP
		// can't reprocess the same unknown event forever.
		s.logger.Info("unknown webhook type", "type", webhook.Type)
	}

	return nil
}

// HandleAuthnWebhook processes an authentication webhook and logs the provider and data.
func (s *WebhookService) HandleAuthnWebhook(ctx context.Context, payload port.AuthnWebhookPayload) error {
	// Authn webhooks (Clerk / Cognito) carry user identifiers and event
	// metadata — logging the raw payload at INFO would put PII in log
	// aggregation. Log only the routing fields.
	s.logger.Info("HandleAuthnWebhook", "provider", payload.Provider, "size", len(payload.Data))

	// Add your business logic here, e.g., validating the payload or triggering workflows.

	return nil
}
