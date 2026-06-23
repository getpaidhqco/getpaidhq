package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// DunningService is the narrow, engine-agnostic dunning aggregate root.
//
// HTTP handlers depend on DunningOrchestrationService (which wraps this one
// and adds engine signaling). Hatchet steps depend on this directly via
// port.DunningService so they don't drag the engine through construction.
type DunningService struct {
	dunningRepository      port.DunningRepository
	subscriptionRepository port.SubscriptionRepository
	customerRepository     port.CustomerRepository
	paymentRepository      port.PaymentRepository
	subscriptionService    port.SubscriptionService
	invoiceService         *InvoiceService
	gatewayFactory         port.GatewayFactory
	pubsub                 port.PubSub
	errorReporter          lib.ErrorReporter
	logger                 port.Logger
}

func NewDunningService(
	dunningRepository port.DunningRepository,
	subscriptionRepository port.SubscriptionRepository,
	customerRepository port.CustomerRepository,
	paymentRepository port.PaymentRepository,
	subscriptionService port.SubscriptionService,
	invoiceService *InvoiceService,
	gatewayFactory port.GatewayFactory,
	pubsub port.PubSub,
	errorReporter lib.ErrorReporter,
	logger port.Logger,
) *DunningService {
	return &DunningService{
		dunningRepository:      dunningRepository,
		subscriptionRepository: subscriptionRepository,
		customerRepository:     customerRepository,
		paymentRepository:      paymentRepository,
		subscriptionService:    subscriptionService,
		invoiceService:         invoiceService,
		gatewayFactory:         gatewayFactory,
		pubsub:                 pubsub,
		errorReporter:          errorReporter,
		logger:                 logger,
	}
}

// ---- Campaigns ----

func (s *DunningService) CreateCampaign(ctx context.Context, input port.CreateDunningCampaignInput) (domain.DunningCampaign, error) {
	s.logger.Info("Creating dunning campaign", "orgId", input.OrgId, "subscriptionId", input.SubscriptionId)

	if _, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.SubscriptionId); err != nil {
		s.logger.Error("Failed to find subscription", err.Error())
		return domain.DunningCampaign{}, lib.NewCustomError(lib.NotFoundError, "Subscription not found", err)
	}
	if _, err := s.customerRepository.FindById(ctx, input.OrgId, input.CustomerId); err != nil {
		s.logger.Error("Failed to find customer", err.Error())
		return domain.DunningCampaign{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	now := time.Now().UTC()
	campaign := domain.DunningCampaign{
		OrgId:                input.OrgId,
		Id:                   lib.GenerateId("dunning_campaign"),
		SubscriptionId:       input.SubscriptionId,
		CustomerId:           input.CustomerId,
		ParentWorkflowId:     input.ParentWorkflowId,
		Status:               domain.DunningStatusActive,
		FailedAmount:         input.FailedAmount,
		Currency:             input.Currency,
		InitialFailureReason: input.InitialFailureReason,
		StartedAt:            now,
		ConfigSnapshot:       input.ConfigSnapshot,
		Metadata:             input.Metadata,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	created, err := s.dunningRepository.CreateCampaign(ctx, campaign)
	if err != nil {
		s.logger.Error("Failed to create dunning campaign", err.Error())
		return domain.DunningCampaign{}, lib.NewCustomError(lib.InternalError, "Failed to create dunning campaign", err)
	}

	_ = s.pubsub.Publish(created.OrgId, port.TopicDunningCampaignStarted, port.NewDunningCampaignEvent(created))
	return created, nil
}

func (s *DunningService) FindCampaignById(ctx context.Context, orgId, id string) (domain.DunningCampaign, error) {
	c, err := s.dunningRepository.FindCampaignById(ctx, orgId, id)
	if err != nil {
		return domain.DunningCampaign{}, lib.NewCustomError(lib.NotFoundError, "Dunning campaign not found", err)
	}
	return c, nil
}

func (s *DunningService) ListCampaigns(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	return s.dunningRepository.FindCampaigns(ctx, orgId, p)
}

func (s *DunningService) ListCampaignsBySubscription(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	return s.dunningRepository.FindCampaignsBySubscriptionId(ctx, orgId, subscriptionId, p)
}

func (s *DunningService) ListCampaignsByCustomer(ctx context.Context, orgId, customerId string, p domain.Pagination) ([]domain.DunningCampaign, int, error) {
	return s.dunningRepository.FindCampaignsByCustomerId(ctx, orgId, customerId, p)
}

func (s *DunningService) PauseCampaign(ctx context.Context, input port.PauseDunningCampaignInput) (domain.DunningCampaign, error) {
	c, err := s.FindCampaignById(ctx, input.OrgId, input.CampaignId)
	if err != nil {
		return domain.DunningCampaign{}, err
	}
	if c.Status != domain.DunningStatusActive {
		return c, lib.NewCustomError(lib.BadRequestError, "Only active campaigns can be paused", nil)
	}
	c.Status = domain.DunningStatusPaused
	c.UpdatedAt = time.Now().UTC()
	updated, err := s.dunningRepository.UpdateCampaign(ctx, c)
	if err != nil {
		return domain.DunningCampaign{}, lib.NewCustomError(lib.InternalError, "Failed to pause campaign", err)
	}
	_ = s.pubsub.Publish(updated.OrgId, port.TopicDunningCampaignPaused, port.NewDunningCampaignEvent(updated))
	return updated, nil
}

func (s *DunningService) ResumeCampaign(ctx context.Context, input port.ResumeDunningCampaignInput) (domain.DunningCampaign, error) {
	c, err := s.FindCampaignById(ctx, input.OrgId, input.CampaignId)
	if err != nil {
		return domain.DunningCampaign{}, err
	}
	if c.Status != domain.DunningStatusPaused {
		return c, lib.NewCustomError(lib.BadRequestError, "Only paused campaigns can be resumed", nil)
	}
	c.Status = domain.DunningStatusActive
	c.UpdatedAt = time.Now().UTC()
	updated, err := s.dunningRepository.UpdateCampaign(ctx, c)
	if err != nil {
		return domain.DunningCampaign{}, lib.NewCustomError(lib.InternalError, "Failed to resume campaign", err)
	}
	_ = s.pubsub.Publish(updated.OrgId, port.TopicDunningCampaignResumed, port.NewDunningCampaignEvent(updated))
	return updated, nil
}

func (s *DunningService) CancelCampaign(ctx context.Context, input port.CancelDunningCampaignInput) (domain.DunningCampaign, error) {
	c, err := s.FindCampaignById(ctx, input.OrgId, input.CampaignId)
	if err != nil {
		return domain.DunningCampaign{}, err
	}
	if c.Status != domain.DunningStatusActive && c.Status != domain.DunningStatusPaused {
		return c, lib.NewCustomError(lib.BadRequestError, "Only active or paused campaigns can be cancelled", nil)
	}
	now := time.Now().UTC()
	c.Status = domain.DunningStatusCancelled
	c.CompletedAt = now
	c.FinalFailureReason = input.Reason
	c.UpdatedAt = now
	updated, err := s.dunningRepository.UpdateCampaign(ctx, c)
	if err != nil {
		return domain.DunningCampaign{}, lib.NewCustomError(lib.InternalError, "Failed to cancel campaign", err)
	}
	_ = s.pubsub.Publish(updated.OrgId, port.TopicDunningCampaignCancelled, port.NewDunningCampaignEvent(updated))
	return updated, nil
}

func (s *DunningService) UpdateCampaign(ctx context.Context, campaign domain.DunningCampaign) (domain.DunningCampaign, error) {
	campaign.UpdatedAt = time.Now().UTC()
	updated, err := s.dunningRepository.UpdateCampaign(ctx, campaign)
	if err != nil {
		return domain.DunningCampaign{}, lib.NewCustomError(lib.InternalError, "Failed to update campaign", err)
	}
	return updated, nil
}

func (s *DunningService) MarkCampaignRecovered(ctx context.Context, orgId, campaignId, recoveryMethod string, recoveredAmount int64) (domain.DunningCampaign, error) {
	c, err := s.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		return domain.DunningCampaign{}, err
	}
	now := time.Now().UTC()
	c.Status = domain.DunningStatusRecovered
	c.RecoveryMethod = recoveryMethod
	c.RecoveredAmount = recoveredAmount
	c.RecoveredAt = now
	c.CompletedAt = now
	c.UpdatedAt = now
	updated, err := s.dunningRepository.UpdateCampaign(ctx, c)
	if err != nil {
		return domain.DunningCampaign{}, lib.NewCustomError(lib.InternalError, "Failed to mark campaign recovered", err)
	}
	_ = s.pubsub.Publish(updated.OrgId, port.TopicDunningCampaignRecovered, port.NewDunningCampaignEvent(updated))
	return updated, nil
}

func (s *DunningService) MarkCampaignFailed(ctx context.Context, orgId, campaignId, finalFailureReason string) (domain.DunningCampaign, error) {
	c, err := s.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		return domain.DunningCampaign{}, err
	}
	now := time.Now().UTC()
	c.Status = domain.DunningStatusFailed
	c.FinalFailureReason = finalFailureReason
	c.CompletedAt = now
	c.UpdatedAt = now
	updated, err := s.dunningRepository.UpdateCampaign(ctx, c)
	if err != nil {
		return domain.DunningCampaign{}, lib.NewCustomError(lib.InternalError, "Failed to mark campaign failed", err)
	}
	_ = s.pubsub.Publish(updated.OrgId, port.TopicDunningCampaignFailed, port.NewDunningCampaignEvent(updated))
	return updated, nil
}

// FailCampaignAndCancelSubscription cancels the subscription, then marks the
// campaign failed. Used by the runner when retries exhaust without an explicit
// CancelAfterAttempt threshold cancelling the subscription first — without
// this the subscription would be left Active with no successful charges ever
// going through.
func (s *DunningService) FailCampaignAndCancelSubscription(ctx context.Context, orgId, campaignId, finalFailureReason string) (domain.DunningCampaign, error) {
	campaign, err := s.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		return domain.DunningCampaign{}, err
	}
	if subscription, err := s.subscriptionRepository.FindById(ctx, orgId, campaign.SubscriptionId); err == nil {
		if subscription.Status != domain.SubscriptionStatusCancelled {
			subscription.Status = domain.SubscriptionStatusCancelled
			subscription.CancelledAt = time.Now().UTC()
			if _, err := s.subscriptionRepository.Update(ctx, subscription); err != nil {
				s.logger.Error("Failed to cancel subscription on dunning exhaustion", "err", err.Error())
			}
		}
		s.writeOffCurrentInvoice(ctx, subscription)
	} else {
		s.logger.Error("Subscription not found while exhausting dunning", "err", err.Error())
	}
	return s.MarkCampaignFailed(ctx, orgId, campaignId, finalFailureReason)
}

// ---- Attempts ----

func (s *DunningService) ListAttemptsByCampaign(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.DunningAttempt, int, error) {
	return s.dunningRepository.FindAttemptsByCampaignId(ctx, orgId, campaignId, p)
}

// TriggerManualAttempt is the same flow as ExecuteAttempt but recorded with
// AttemptType=manual and a TriggeredBy stamp from the caller. Used by the
// admin HTTP endpoint and the payment-method-updated signal handler.
func (s *DunningService) TriggerManualAttempt(ctx context.Context, input port.TriggerManualAttemptInput) (domain.DunningAttempt, error) {
	s.logger.Info("Triggering manual dunning attempt", "campaignId", input.CampaignId)
	attempt, err := s.runChargeAttempt(ctx, input.OrgId, input.CampaignId, domain.DunningAttemptTypeManual, input.TriggeredBy, input.PaymentMethodId)
	if err != nil {
		return domain.DunningAttempt{}, err
	}
	if attempt.Status == domain.PaymentStatusSucceeded {
		if _, err := s.MarkCampaignRecovered(ctx, attempt.OrgId, attempt.DunningCampaignId, "manual_retry", attempt.Amount); err != nil {
			s.logger.Error("Failed to mark campaign recovered after manual attempt", "err", err.Error())
		}
	}
	return attempt, nil
}

// ExecuteAttempt is the engine-side entrypoint: each scheduled retry from the
// runner becomes one call here.
func (s *DunningService) ExecuteAttempt(ctx context.Context, orgId, campaignId string, attemptType domain.DunningAttemptType) (domain.DunningAttempt, error) {
	s.logger.Info("Executing dunning attempt", "campaignId", campaignId, "type", string(attemptType))
	return s.runChargeAttempt(ctx, orgId, campaignId, attemptType, string(attemptType), "")
}

func (s *DunningService) runChargeAttempt(ctx context.Context, orgId, campaignId string, attemptType domain.DunningAttemptType, triggeredBy, paymentMethodIdOverride string) (domain.DunningAttempt, error) {
	campaign, err := s.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		return domain.DunningAttempt{}, err
	}
	subscription, err := s.subscriptionRepository.FindById(ctx, orgId, campaign.SubscriptionId)
	if err != nil {
		return domain.DunningAttempt{}, lib.NewCustomError(lib.NotFoundError, "Subscription not found", err)
	}
	customer, err := s.subscriptionService.GetSubscriptionCustomer(ctx, subscription)
	if err != nil {
		return domain.DunningAttempt{}, lib.NewCustomError(lib.InternalError, "Failed to load customer", err)
	}

	pmId := subscription.PaymentMethodId
	if paymentMethodIdOverride != "" {
		pmId = paymentMethodIdOverride
	}
	paymentMethod, err := s.customerRepository.FindPaymentMethodById(ctx, orgId, pmId)
	if err != nil {
		return domain.DunningAttempt{}, lib.NewCustomError(lib.NotFoundError, "Payment method not found", err)
	}

	gw, err := s.gatewayFactory.NewGateway(ctx, orgId, string(subscription.PspId))
	if err != nil {
		return domain.DunningAttempt{}, lib.NewCustomError(lib.InternalError, "Failed to get gateway", err)
	}

	startedAt := time.Now().UTC()
	chargeResult := gw.ChargePayment(ctx, port.ChargePaymentInput{
		OrgId:          orgId,
		OrderId:        subscription.OrderId,
		SubscriptionId: subscription.Id,
		Amount:         campaign.FailedAmount,
		Currency:       campaign.Currency,
		PaymentMethod: port.PspPaymentMethod{
			PspId:       paymentMethod.Id,
			Name:        paymentMethod.Name,
			Type:        string(paymentMethod.Type),
			IsRecurring: true,
			Token:       paymentMethod.Token.Reveal(),
		},
		Customer: customer,
	})
	completedAt := time.Now().UTC()

	status := domain.PaymentStatusFailed
	switch chargeResult.Status {
	case port.ChargePaymentStatusSuccess:
		status = domain.PaymentStatusSucceeded
	case port.ChargePaymentStatusPending:
		status = domain.PaymentStatusPending
	case port.ChargePaymentStatusError:
		status = domain.PaymentStatusFailed
	case port.ChargePaymentStatusGatewayError:
		status = domain.PaymentStatusFailed
	}

	var processorResponse map[string]any
	if chargeResult.PspResponse != nil {
		if b, err := json.Marshal(chargeResult.PspResponse); err == nil {
			_ = json.Unmarshal(b, &processorResponse)
		}
	}

	attempt := domain.DunningAttempt{
		OrgId:             orgId,
		Id:                lib.GenerateId("dunning_attempt"),
		DunningCampaignId: campaign.Id,
		SubscriptionId:    subscription.Id,
		AttemptNumber:     campaign.TotalAttempts + 1,
		AttemptType:       attemptType,
		Amount:            chargeResult.AmountCharged,
		Currency:          string(chargeResult.Currency),
		PaymentMethodId:   paymentMethod.Id,
		Status:            status,
		FailureReason:     chargeResult.ErrorReason,
		FailureCode:       chargeResult.ErrorCode,
		ProcessorResponse: processorResponse,
		ProcessingTimeMs:  int(completedAt.Sub(startedAt).Milliseconds()),
		AttemptedAt:       startedAt,
		CompletedAt:       completedAt,
		TriggeredBy:       triggeredBy,
		CreatedAt:         completedAt,
	}
	created, err := s.dunningRepository.CreateAttempt(ctx, attempt)
	if err != nil {
		return domain.DunningAttempt{}, lib.NewCustomError(lib.InternalError, "Failed to record attempt", err)
	}

	// Bump campaign counters; recovery / suspension decisions are made by the
	// runner via UpdateCampaignWithAttemptResult so escalation policy lives in
	// one place.
	campaign.TotalAttempts++
	switch attemptType {
	case domain.DunningAttemptTypeImmediate:
		campaign.ImmediateAttempts++
	case domain.DunningAttemptTypeProgressive:
		campaign.ProgressiveAttempts++
	}
	campaign.LastAttemptAt = completedAt
	campaign.UpdatedAt = completedAt
	if _, err := s.dunningRepository.UpdateCampaign(ctx, campaign); err != nil {
		s.logger.Error("Failed to update campaign attempt counters", "err", err.Error())
	}

	if status == domain.PaymentStatusSucceeded {
		_ = s.pubsub.Publish(orgId, port.TopicDunningAttemptSucceeded, port.NewDunningAttemptEvent(created, campaign.CustomerId, false, false))
	} else {
		_ = s.pubsub.Publish(orgId, port.TopicDunningAttemptFailed, port.NewDunningAttemptEvent(created, campaign.CustomerId, false, false))
	}
	return created, nil
}

// UpdateCampaignWithAttemptResult is called by the engine adapter after each
// attempt; it owns the escalation policy (suspend / cancel / mark failed) so
// the workflow body only needs to know "did this finish the campaign?".
func (s *DunningService) UpdateCampaignWithAttemptResult(ctx context.Context, attempt domain.DunningAttempt, config domain.DunningConfig, attemptContext domain.DunningAttemptContext) (domain.DunningCampaign, error) {
	campaign, err := s.FindCampaignById(ctx, attempt.OrgId, attempt.DunningCampaignId)
	if err != nil {
		return domain.DunningCampaign{}, err
	}

	if attempt.Status == domain.PaymentStatusSucceeded {
		updated, err := s.MarkCampaignRecovered(ctx, campaign.OrgId, campaign.Id, "retry_recovered", attempt.Amount)
		if err != nil {
			return domain.DunningCampaign{}, err
		}
		// Reactivate the subscription if dunning had suspended it. We look up
		// the live status rather than trusting the runner's hint — the hint
		// says "the runner thinks it crossed the suspension threshold," not
		// "the subscription is in fact suspended." We publish the real
		// OldStatus so consumers see an honest transition.
		if attemptContext.WasSubscriptionSuspended {
			subscription, err := s.subscriptionRepository.FindById(ctx, campaign.OrgId, campaign.SubscriptionId)
			if err == nil && subscription.Status != domain.SubscriptionStatusActive {
				oldStatus := subscription.Status
				subscription.Status = domain.SubscriptionStatusActive
				if _, err := s.subscriptionRepository.Update(ctx, subscription); err != nil {
					s.logger.Error("Failed to reactivate subscription", "err", err.Error())
				} else {
					_ = s.pubsub.Publish(campaign.OrgId, port.TopicDunningSubscriptionReactivated, port.DunningSubscriptionEvent{
						OrgId:          campaign.OrgId,
						CampaignId:     campaign.Id,
						SubscriptionId: subscription.Id,
						CustomerId:     campaign.CustomerId,
						OldStatus:      oldStatus,
						NewStatus:      domain.SubscriptionStatusActive,
					})
				}
			}
		}
		return updated, nil
	}

	// Failure path: escalate if we've crossed the cancel threshold, else continue.
	if config.EscalationRules.CancelAfterAttempt > 0 && attemptContext.AttemptNumber >= config.EscalationRules.CancelAfterAttempt {
		// Cancel the subscription and mark the campaign failed.
		if subscription, err := s.subscriptionRepository.FindById(ctx, campaign.OrgId, campaign.SubscriptionId); err == nil {
			subscription.Status = domain.SubscriptionStatusCancelled
			subscription.CancelledAt = time.Now().UTC()
			if _, err := s.subscriptionRepository.Update(ctx, subscription); err != nil {
				s.logger.Error("Failed to cancel subscription after dunning exhaustion", "err", err.Error())
			}
			s.writeOffCurrentInvoice(ctx, subscription)
		}
		return s.MarkCampaignFailed(ctx, campaign.OrgId, campaign.Id, "max_attempts_reached")
	}

	isFinalNotice := config.EscalationRules.FinalNoticeAttempt > 0 && attemptContext.AttemptNumber >= config.EscalationRules.FinalNoticeAttempt
	shouldSuspend := config.EscalationRules.SuspendAfterAttempt > 0 && attemptContext.AttemptNumber >= config.EscalationRules.SuspendAfterAttempt

	// Persist the suspension the first time we cross the threshold. The
	// shouldSuspend flag is "should be suspended by now" — load the
	// subscription and only flip it if it isn't already Unpaid (or in a
	// terminal status that would make suspension nonsensical).
	if shouldSuspend {
		if subscription, err := s.subscriptionRepository.FindById(ctx, campaign.OrgId, campaign.SubscriptionId); err == nil {
			if subscription.Status != domain.SubscriptionStatusUnpaid &&
				subscription.Status != domain.SubscriptionStatusCancelled &&
				subscription.Status != domain.SubscriptionStatusExpired &&
				subscription.Status != domain.SubscriptionStatusCompleted {
				oldStatus := subscription.Status
				subscription.Status = domain.SubscriptionStatusUnpaid
				if _, err := s.subscriptionRepository.Update(ctx, subscription); err != nil {
					s.logger.Error("Failed to suspend subscription during dunning", "err", err.Error())
				} else {
					_ = s.pubsub.Publish(campaign.OrgId, port.TopicDunningSubscriptionSuspended, port.DunningSubscriptionEvent{
						OrgId:          campaign.OrgId,
						CampaignId:     campaign.Id,
						SubscriptionId: subscription.Id,
						CustomerId:     campaign.CustomerId,
						OldStatus:      oldStatus,
						NewStatus:      domain.SubscriptionStatusUnpaid,
					})
					s.writeOffCurrentInvoice(ctx, subscription)
				}
			}
		}
	}

	_ = s.pubsub.Publish(campaign.OrgId, port.TopicDunningAttemptFailed, port.NewDunningAttemptEvent(attempt, campaign.CustomerId, shouldSuspend, isFinalNotice))
	return campaign, nil
}

// ---- Communications ----

func (s *DunningService) ListCommunicationsByCampaign(ctx context.Context, orgId, campaignId string, p domain.Pagination) ([]domain.DunningCommunication, int, error) {
	return s.dunningRepository.FindCommunicationsByCampaignId(ctx, orgId, campaignId, p)
}

// SendCommunication just publishes the event for the notifications service to
// pick up and dispatch. The communication row will be inserted by the
// notification side once the message is sent.
func (s *DunningService) SendCommunication(ctx context.Context, orgId, campaignId string, attemptNumber int) error {
	campaign, err := s.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		return err
	}
	s.logger.Info("Publishing dunning communication request", "campaignId", campaignId, "attemptNumber", attemptNumber)
	return s.pubsub.Publish(orgId, port.TopicDunningCommunicationSent, port.DunningCommunicationEvent{
		OrgId:         orgId,
		CampaignId:    campaign.Id,
		CustomerId:    campaign.CustomerId,
		AttemptNumber: attemptNumber,
		Status:        domain.CommunicationStatusPending,
	})
}

// ---- Tokens ----

func (s *DunningService) CreatePaymentUpdateToken(ctx context.Context, input port.CreatePaymentUpdateTokenInput) (domain.PaymentUpdateToken, error) {
	s.logger.Info("Creating payment update token", "subscriptionId", input.SubscriptionId)

	if _, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.SubscriptionId); err != nil {
		return domain.PaymentUpdateToken{}, lib.NewCustomError(lib.NotFoundError, "Subscription not found", err)
	}
	if _, err := s.customerRepository.FindById(ctx, input.OrgId, input.CustomerId); err != nil {
		return domain.PaymentUpdateToken{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}

	maxUses := input.MaxUses
	if maxUses <= 0 {
		maxUses = 5
	}
	expiryHours := input.ExpiryHours
	if expiryHours <= 0 {
		expiryHours = 72
	}
	allowed := input.AllowedActions
	if allowed == nil {
		allowed = map[string]bool{"update_payment_method": true, "retry_payment": true}
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(expiryHours) * time.Hour)
	token := domain.PaymentUpdateToken{
		OrgId:             input.OrgId,
		TokenId:           lib.GenerateId("payment_update_token"),
		SubscriptionId:    input.SubscriptionId,
		CustomerId:        input.CustomerId,
		DunningCampaignId: input.DunningCampaignId,
		TokenData: map[string]any{
			"expires_at":      expiresAt.Format(time.RFC3339),
			"max_uses":        maxUses,
			"allowed_actions": allowed,
		},
		Signature:      "",
		ExpiresAt:      expiresAt,
		MaxUses:        maxUses,
		UsedCount:      0,
		Status:         domain.TokenStatusActive,
		AllowedActions: allowed,
		AdminGenerated: input.AdminGenerated,
		AdminUserId:    input.AdminUserId,
		AdminReason:    input.AdminReason,
		AdminNotes:     input.AdminNotes,
		CreatedBy:      input.CreatedBy,
		CreatedAt:      now,
	}
	created, err := s.dunningRepository.CreateToken(ctx, token)
	if err != nil {
		return domain.PaymentUpdateToken{}, lib.NewCustomError(lib.InternalError, "Failed to create token", err)
	}
	_ = s.pubsub.Publish(created.OrgId, port.TopicDunningTokenCreated, port.NewDunningTokenEvent(created))
	return created, nil
}

func (s *DunningService) VerifyPaymentUpdateToken(ctx context.Context, orgId, tokenId string) (domain.PaymentUpdateToken, error) {
	t, err := s.dunningRepository.FindTokenById(ctx, orgId, tokenId)
	if err != nil {
		return domain.PaymentUpdateToken{}, lib.NewCustomError(lib.NotFoundError, "Token not found", err)
	}
	if t.Status != domain.TokenStatusActive {
		return t, lib.NewCustomError(lib.BadRequestError, "Token is not active", nil)
	}
	if time.Now().UTC().After(t.ExpiresAt) {
		t.Status = domain.TokenStatusExpired
		_, _ = s.dunningRepository.UpdateToken(ctx, t)
		return t, lib.NewCustomError(lib.BadRequestError, "Token has expired", nil)
	}
	if t.UsedCount >= t.MaxUses {
		t.Status = domain.TokenStatusMaxUsesReached
		_, _ = s.dunningRepository.UpdateToken(ctx, t)
		return t, lib.NewCustomError(lib.BadRequestError, "Token has reached max uses", nil)
	}
	return t, nil
}

func (s *DunningService) ActivatePaymentUpdateToken(ctx context.Context, input port.ActivatePaymentUpdateTokenInput) (domain.PaymentUpdateToken, error) {
	t, err := s.VerifyPaymentUpdateToken(ctx, input.OrgId, input.TokenId)
	if err != nil {
		return domain.PaymentUpdateToken{}, err
	}
	now := time.Now().UTC()
	t.UsedCount++
	t.LastUsedAt = now
	t.LastUsedIp = input.UsedIp
	if t.UsedCount >= t.MaxUses {
		t.Status = domain.TokenStatusMaxUsesReached
	}
	updated, err := s.dunningRepository.UpdateToken(ctx, t)
	if err != nil {
		return domain.PaymentUpdateToken{}, lib.NewCustomError(lib.InternalError, "Failed to update token", err)
	}
	_ = s.pubsub.Publish(updated.OrgId, port.TopicDunningTokenActivated, port.NewDunningTokenEvent(updated))
	return updated, nil
}

func (s *DunningService) RevokePaymentUpdateToken(ctx context.Context, orgId, tokenId string) (domain.PaymentUpdateToken, error) {
	t, err := s.dunningRepository.FindTokenById(ctx, orgId, tokenId)
	if err != nil {
		return domain.PaymentUpdateToken{}, lib.NewCustomError(lib.NotFoundError, "Token not found", err)
	}
	if t.Status != domain.TokenStatusActive {
		return t, lib.NewCustomError(lib.BadRequestError, "Only active tokens can be revoked", nil)
	}
	t.Status = domain.TokenStatusRevoked
	updated, err := s.dunningRepository.UpdateToken(ctx, t)
	if err != nil {
		return domain.PaymentUpdateToken{}, lib.NewCustomError(lib.InternalError, "Failed to revoke token", err)
	}
	_ = s.pubsub.Publish(updated.OrgId, port.TopicDunningTokenRevoked, port.NewDunningTokenEvent(updated))
	return updated, nil
}

// ---- Configurations ----

func (s *DunningService) CreateConfiguration(ctx context.Context, input port.CreateDunningConfigurationInput) (domain.DunningConfiguration, error) {
	configMap, err := configToMap(input.Config)
	if err != nil {
		return domain.DunningConfiguration{}, lib.NewCustomError(lib.BadRequestError, "Invalid dunning config", err)
	}
	now := time.Now().UTC()
	cfg := domain.DunningConfiguration{
		OrgId:            input.OrgId,
		Id:               lib.GenerateId("dunning_configuration"),
		Name:             input.Name,
		Description:      input.Description,
		Priority:         input.Priority,
		AppliesTo:        input.AppliesTo,
		TargetRules:      input.TargetRules,
		Config:           configMap,
		Status:           domain.ConfigStatusActive,
		IsAbTest:         input.IsAbTest,
		AbTestPercentage: input.AbTestPercentage,
		CreatedBy:        input.CreatedBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	created, err := s.dunningRepository.CreateConfiguration(ctx, cfg)
	if err != nil {
		return domain.DunningConfiguration{}, lib.NewCustomError(lib.InternalError, "Failed to create configuration", err)
	}
	_ = s.pubsub.Publish(created.OrgId, port.TopicDunningConfigurationCreated, port.NewDunningConfigurationEvent(created))
	return created, nil
}

func (s *DunningService) GetConfiguration(ctx context.Context, orgId, id string) (domain.DunningConfiguration, error) {
	c, err := s.dunningRepository.FindConfigurationById(ctx, orgId, id)
	if err != nil {
		return domain.DunningConfiguration{}, lib.NewCustomError(lib.NotFoundError, "Configuration not found", err)
	}
	return c, nil
}

func (s *DunningService) ListConfigurations(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningConfiguration, int, error) {
	return s.dunningRepository.FindConfigurations(ctx, orgId, p)
}

func (s *DunningService) UpdateConfiguration(ctx context.Context, input port.UpdateDunningConfigurationInput) (domain.DunningConfiguration, error) {
	c, err := s.GetConfiguration(ctx, input.OrgId, input.Id)
	if err != nil {
		return domain.DunningConfiguration{}, err
	}
	if input.Name != "" {
		c.Name = input.Name
	}
	if input.Description != "" {
		c.Description = input.Description
	}
	if input.Priority != 0 {
		c.Priority = input.Priority
	}
	if input.AppliesTo != "" {
		c.AppliesTo = input.AppliesTo
	}
	if input.TargetRules != nil {
		c.TargetRules = input.TargetRules
	}
	if input.Config != nil {
		configMap, err := configToMap(*input.Config)
		if err != nil {
			return domain.DunningConfiguration{}, lib.NewCustomError(lib.BadRequestError, "Invalid dunning config", err)
		}
		c.Config = configMap
	}
	if input.Status != "" {
		c.Status = input.Status
	}
	if input.IsAbTest != nil {
		c.IsAbTest = *input.IsAbTest
	}
	if input.AbTestPercentage != nil {
		c.AbTestPercentage = *input.AbTestPercentage
	}
	c.UpdatedAt = time.Now().UTC()
	updated, err := s.dunningRepository.UpdateConfiguration(ctx, c)
	if err != nil {
		return domain.DunningConfiguration{}, lib.NewCustomError(lib.InternalError, "Failed to update configuration", err)
	}
	_ = s.pubsub.Publish(updated.OrgId, port.TopicDunningConfigurationUpdated, port.NewDunningConfigurationEvent(updated))
	return updated, nil
}

// ResolveConfig picks the highest-priority active configuration for the org
// and decodes its Config blob back into a typed DunningConfig. Falls back to
// the default policy if none match.
func (s *DunningService) ResolveConfig(ctx context.Context, orgId string) (domain.DunningConfig, error) {
	configs, err := s.dunningRepository.FindConfigurationsByPriority(ctx, orgId)
	if err != nil {
		s.logger.Error("Failed to list configurations, using default", "err", err.Error())
		return domain.DefaultDunningConfig(), nil
	}
	for _, c := range configs {
		if cfg, err := configFromMap(c.Config); err == nil {
			return cfg, nil
		} else {
			s.logger.Error("Failed to decode dunning config, skipping", "configId", c.Id, "err", err.Error())
		}
	}
	return domain.DefaultDunningConfig(), nil
}

// LoadConfigForCampaign returns the dunning config the campaign was started
// under. Prefers the snapshot persisted on the campaign at creation time so
// mid-flight config edits don't change cadence for already-running campaigns.
// Falls back to ResolveConfig for campaigns started before snapshotting
// existed.
func (s *DunningService) LoadConfigForCampaign(ctx context.Context, orgId, campaignId string) (domain.DunningConfig, error) {
	campaign, err := s.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		return domain.DunningConfig{}, err
	}
	if len(campaign.ConfigSnapshot) > 0 {
		if cfg, err := configFromMap(campaign.ConfigSnapshot); err == nil {
			return cfg, nil
		} else {
			s.logger.Error("Failed to decode campaign config snapshot, falling back to live config", "campaignId", campaignId, "err", err.Error())
		}
	}
	return s.ResolveConfig(ctx, orgId)
}

// ---- Customer history ----

func (s *DunningService) GetCustomerDunningHistory(ctx context.Context, orgId, customerId string) (domain.CustomerDunningHistory, error) {
	if _, err := s.customerRepository.FindById(ctx, orgId, customerId); err != nil {
		return domain.CustomerDunningHistory{}, lib.NewCustomError(lib.NotFoundError, "Customer not found", err)
	}
	return s.dunningRepository.GetCustomerDunningHistory(ctx, orgId, customerId)
}

// writeOffCurrentInvoice marks the subscription's current-cycle invoice
// uncollectible when dunning ends collection. No-op if absent/terminal or if
// no invoice service is wired (some unit tests).
func (s *DunningService) writeOffCurrentInvoice(ctx context.Context, sub domain.Subscription) {
	if s.invoiceService == nil {
		return
	}
	inv, err := s.invoiceService.FindCurrentCycle(ctx, sub.OrgId, sub.Id, sub.CyclesProcessed)
	if err != nil {
		return
	}
	if inv.Status != domain.InvoiceStatusOpen && inv.Status != domain.InvoiceStatusDraft {
		return
	}
	if _, err := s.invoiceService.MarkUncollectible(ctx, sub.OrgId, inv.Id); err != nil {
		s.logger.Error("Failed to mark invoice uncollectible on dunning exhaustion", "err", err.Error())
	}
}

// ---- helpers ----

func configToMap(c domain.DunningConfig) (map[string]any, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return m, nil
}

func configFromMap(m map[string]any) (domain.DunningConfig, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return domain.DunningConfig{}, fmt.Errorf("marshal map: %w", err)
	}
	var c domain.DunningConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return domain.DunningConfig{}, fmt.Errorf("unmarshal config: %w", err)
	}
	return c, nil
}
