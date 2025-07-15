package services

import (
	"context"
	"encoding/json"
	"errors"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

// DunningService implements the interfaces.DunningService interface
type DunningService struct {
	dunningRepository      repositories.DunningRepository
	subscriptionRepository repositories.SubscriptionRepository
	customerRepository     repositories.CustomerRepository
	paymentRepository      repositories.PaymentRepository
	subscriptionService    interfaces.SubscriptionService
	gatewayFactory         factories.GatewayFactory
	pubsub                 events.NotificationPublisher
	logger                 logger.Logger
	errorReporter          lib.ErrorReporter
}

// NewDunningService creates a new DunningService
func NewDunningService(
	dunningRepository repositories.DunningRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	customerRepository repositories.CustomerRepository,
	paymentRepository repositories.PaymentRepository,
	subscriptionService interfaces.SubscriptionService,
	gatewayFactory factories.GatewayFactory,
	pubsub events.NotificationPublisher,
	logger logger.Logger,
	errorReporter lib.ErrorReporter,
) interfaces.DunningService {
	return &DunningService{
		dunningRepository:      dunningRepository,
		subscriptionRepository: subscriptionRepository,
		customerRepository:     customerRepository,
		paymentRepository:      paymentRepository,
		subscriptionService:    subscriptionService,
		gatewayFactory:         gatewayFactory,
		pubsub:                 pubsub,
		logger:                 logger,
		errorReporter:          errorReporter,
	}
}

// CreateCampaign creates a new dunning campaign
func (s *DunningService) CreateCampaign(ctx context.Context, input dto.CreateDunningCampaignInput) (dunning.DunningCampaign, error) {
	s.logger.Info("Creating dunning campaign", "orgId", input.OrgId, "subscriptionId", input.SubscriptionId)

	// Validate subscription exists
	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.SubscriptionId)
	if err != nil {
		s.logger.Error("Failed to find subscription", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Validate customer exists
	_, err = s.customerRepository.FindById(ctx, input.OrgId, input.CustomerId)
	if err != nil {
		s.logger.Error("Failed to find customer", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Create campaign
	campaign := dunning.DunningCampaign{
		OrgId:                input.OrgId,
		Id:                   lib.GenerateId("dun"),
		SubscriptionId:       input.SubscriptionId,
		CustomerId:           input.CustomerId,
		ParentWorkflowId:     input.ParentWorkflowId,
		Status:               dunning.DunningStatusActive,
		FailedAmount:         input.FailedAmount,
		Currency:             input.Currency,
		InitialFailureReason: input.InitialFailureReason,
		TotalAttempts:        0,
		ImmediateAttempts:    0,
		ProgressiveAttempts:  0,
		StartedAt:            time.Now().UTC(),
		Metadata:             input.Metadata,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	// Save campaign
	campaign, err = s.dunningRepository.CreateCampaign(ctx, campaign)
	if err != nil {
		s.logger.Error("Failed to create dunning campaign", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Update subscription
	subscription.DunningActive = true
	subscription.ActiveDunningCampaignId = campaign.Id
	_, err = s.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		s.logger.Error("Failed to update subscription", err.Error())
		// Continue anyway, as the campaign has already been created
	}

	// Publish event
	event := topic.NewDunningCampaignEvent(campaign)
	err = s.pubsub.Publish(campaign.OrgId, topic.DunningCampaignStarted, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning campaign started event", err.Error())
		// Continue anyway, as the campaign has already been created
	}

	return campaign, nil
}

// FindCampaignById finds a dunning campaign by ID
func (s *DunningService) FindCampaignById(ctx context.Context, orgId string, id string) (dunning.DunningCampaign, error) {
	s.logger.Info("Finding dunning campaign", "orgId", orgId, "id", id)

	campaign, err := s.dunningRepository.FindCampaignById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find dunning campaign", err.Error())
		return dunning.DunningCampaign{}, err
	}

	return campaign, nil
}

// ListCampaigns lists dunning campaigns
func (s *DunningService) ListCampaigns(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	s.logger.Info("Listing dunning campaigns", "orgId", orgId)

	campaigns, total, err := s.dunningRepository.FindCampaigns(ctx, orgId, entities.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortDirection: pagination.SortDirection,
		SortBy:        pagination.SortBy,
	})
	if err != nil {
		s.logger.Error("Failed to list dunning campaigns", err.Error())
		return nil, 0, err
	}

	return campaigns, total, nil
}

// ListCampaignsBySubscription lists dunning campaigns by subscription ID
func (s *DunningService) ListCampaignsBySubscription(ctx context.Context, orgId string, subscriptionId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	s.logger.Info("Listing dunning campaigns by subscription", "orgId", orgId, "subscriptionId", subscriptionId)

	campaigns, total, err := s.dunningRepository.FindCampaignsBySubscriptionId(ctx, orgId, subscriptionId, entities.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortDirection: pagination.SortDirection,
		SortBy:        pagination.SortBy,
	})
	if err != nil {
		s.logger.Error("Failed to list dunning campaigns by subscription", err.Error())
		return nil, 0, err
	}

	return campaigns, total, nil
}

// ListCampaignsByCustomer lists dunning campaigns by customer ID
func (s *DunningService) ListCampaignsByCustomer(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	s.logger.Info("Listing dunning campaigns by customer", "orgId", orgId, "customerId", customerId)

	campaigns, total, err := s.dunningRepository.FindCampaignsByCustomerId(ctx, orgId, customerId, entities.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortDirection: pagination.SortDirection,
		SortBy:        pagination.SortBy,
	})
	if err != nil {
		s.logger.Error("Failed to list dunning campaigns by customer", err.Error())
		return nil, 0, err
	}

	return campaigns, total, nil
}

// PauseCampaign pauses a dunning campaign
func (s *DunningService) PauseCampaign(ctx context.Context, input dto.PauseDunningCampaignInput) (dunning.DunningCampaign, error) {
	s.logger.Info("Pausing dunning campaign", "orgId", input.OrgId, "id", input.Id)

	// Find campaign
	campaign, err := s.dunningRepository.FindCampaignById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find dunning campaign", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Validate campaign status
	if campaign.Status != dunning.DunningStatusActive {
		s.logger.Info("Campaign is not active", "status", campaign.Status)
		return dunning.DunningCampaign{}, lib.NewCustomError(lib.BadRequestError, "campaign is not active", nil)
	}

	// Update campaign
	campaign.Status = dunning.DunningStatusPaused
	campaign.UpdatedAt = time.Now().UTC()

	// Save campaign
	campaign, err = s.dunningRepository.UpdateCampaign(ctx, campaign)
	if err != nil {
		s.logger.Error("Failed to update dunning campaign", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Publish event
	event := topic.NewDunningCampaignEvent(campaign)
	err = s.pubsub.Publish(campaign.OrgId, topic.DunningCampaignPaused, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning campaign paused event", err.Error())
		// Continue anyway, as the campaign has already been updated
	}

	return campaign, nil
}

// ResumeCampaign resumes a paused dunning campaign
func (s *DunningService) ResumeCampaign(ctx context.Context, input dto.ResumeDunningCampaignInput) (dunning.DunningCampaign, error) {
	s.logger.Info("Resuming dunning campaign", "orgId", input.OrgId, "id", input.Id)

	// Find campaign
	campaign, err := s.dunningRepository.FindCampaignById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find dunning campaign", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Validate campaign status
	if campaign.Status != dunning.DunningStatusPaused {
		s.logger.Info("Campaign is not paused", "status", campaign.Status)
		return dunning.DunningCampaign{}, lib.NewCustomError(lib.BadRequestError, "campaign is not paused", nil)
	}

	// Update campaign
	campaign.Status = dunning.DunningStatusActive
	campaign.UpdatedAt = time.Now().UTC()

	// Save campaign
	campaign, err = s.dunningRepository.UpdateCampaign(ctx, campaign)
	if err != nil {
		s.logger.Error("Failed to update dunning campaign", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Publish event
	event := topic.NewDunningCampaignEvent(campaign)
	err = s.pubsub.Publish(campaign.OrgId, topic.DunningCampaignResumed, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning campaign resumed event", err.Error())
		// Continue anyway, as the campaign has already been updated
	}

	return campaign, nil
}

// CancelCampaign cancels a dunning campaign
func (s *DunningService) CancelCampaign(ctx context.Context, input dto.CancelDunningCampaignInput) (dunning.DunningCampaign, error) {
	s.logger.Info("Cancelling dunning campaign", "orgId", input.OrgId, "id", input.Id)

	// Find campaign
	campaign, err := s.dunningRepository.FindCampaignById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find dunning campaign", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Validate campaign status
	if campaign.Status != dunning.DunningStatusActive && campaign.Status != dunning.DunningStatusPaused {
		s.logger.Info("Campaign cannot be cancelled", "status", campaign.Status)
		return dunning.DunningCampaign{}, lib.NewCustomError(lib.BadRequestError, "campaign cannot be cancelled", nil)
	}

	// Update campaign
	campaign.Status = dunning.DunningStatusCancelled
	campaign.CompletedAt = time.Now().UTC()
	campaign.UpdatedAt = time.Now().UTC()
	campaign.FinalFailureReason = input.Reason

	// Save campaign
	campaign, err = s.dunningRepository.UpdateCampaign(ctx, campaign)
	if err != nil {
		s.logger.Error("Failed to update dunning campaign", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Update subscription
	subscription, err := s.subscriptionRepository.FindById(ctx, campaign.OrgId, campaign.SubscriptionId)
	if err != nil {
		s.logger.Error("Failed to find subscription", err.Error())
		// Continue anyway, as the campaign has already been updated
	} else {
		subscription.DunningActive = false
		subscription.ActiveDunningCampaignId = ""
		_, err = s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to update subscription", err.Error())
			// Continue anyway, as the campaign has already been updated
		}
	}

	// Publish event
	event := topic.NewDunningCampaignEvent(campaign)
	err = s.pubsub.Publish(campaign.OrgId, topic.DunningCampaignCancelled, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning campaign cancelled event", err.Error())
		// Continue anyway, as the campaign has already been updated
	}

	return campaign, nil
}

// UpdateCampaign updates a dunning campaign
func (s *DunningService) UpdateCampaign(ctx context.Context, orgId string, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error) {
	s.logger.Info("Updating dunning campaign", "orgId", orgId, "id", campaign.Id)

	// Ensure the campaign belongs to the organization
	if campaign.OrgId != orgId {
		return dunning.DunningCampaign{}, lib.NewCustomError(lib.BadRequestError, "campaign does not belong to organization", nil)
	}

	// Update the campaign
	updatedCampaign, err := s.dunningRepository.UpdateCampaign(ctx, campaign)
	if err != nil {
		s.logger.Error("Failed to update dunning campaign", err.Error())
		return dunning.DunningCampaign{}, err
	}

	return updatedCampaign, nil
}

// ListAttemptsByCampaign lists dunning attempts by campaign ID
func (s *DunningService) ListAttemptsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningAttempt, int, error) {
	s.logger.Info("Listing dunning attempts by campaign", "orgId", orgId, "campaignId", campaignId)

	attempts, total, err := s.dunningRepository.FindAttemptsByCampaignId(ctx, orgId, campaignId, entities.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortDirection: pagination.SortDirection,
		SortBy:        pagination.SortBy,
	})
	if err != nil {
		s.logger.Error("Failed to list dunning attempts by campaign", err.Error())
		return nil, 0, err
	}

	return attempts, total, nil
}

// TriggerManualAttempt triggers a manual payment attempt
func (s *DunningService) TriggerChargeAttempt(ctx context.Context, input dto.TriggerAttemptInput) (dunning.DunningAttempt, error) {
	s.logger.Info("Triggering manual payment attempt", "orgId", input.OrgId, "campaignId", input.CampaignId)

	// Find campaign
	campaign, err := s.dunningRepository.FindCampaignById(ctx, input.OrgId, input.CampaignId)
	if err != nil {
		s.logger.Error("Failed to find dunning campaign", err.Error())
		return dunning.DunningAttempt{}, err
	}

	// Validate campaign status
	if campaign.Status != dunning.DunningStatusActive {
		s.logger.Info("Campaign is not active", "status", campaign.Status)
		return dunning.DunningAttempt{}, lib.NewCustomError(lib.BadRequestError, "campaign is not active", nil)
	}

	// Get the subscription
	subscription, err := s.subscriptionRepository.FindById(ctx, input.OrgId, campaign.SubscriptionId)
	if err != nil {
		s.logger.Error("Failed to find subscription", err.Error())
		return dunning.DunningAttempt{}, err
	}

	// Determine which payment method to use
	var paymentMethodId string
	if input.PaymentMethodId != "" {
		paymentMethodId = input.PaymentMethodId
	} else {
		paymentMethodId = subscription.PaymentMethodId
	}

	// Create attempt
	attempt := dunning.DunningAttempt{
		OrgId:             input.OrgId,
		Id:                lib.GenerateId("dun"),
		DunningCampaignId: input.CampaignId,
		SubscriptionId:    campaign.SubscriptionId,
		AttemptNumber:     campaign.TotalAttempts + 1,
		AttemptType:       input.Type,
		Amount:            int64(campaign.FailedAmount),
		Currency:          campaign.Currency,
		PaymentMethodId:   paymentMethodId,
		AttemptedAt:       time.Now().UTC(),
		TriggeredBy:       input.TriggeredBy,
		CreatedAt:         time.Now().UTC(),
	}

	chargeResult, err := s.subscriptionService.ProcessSubscriptionCharge(ctx, subscription)
	if err != nil {
		return dunning.DunningAttempt{}, err
	}

	// Update attempt with charge result
	attempt.Status = chargeResult.Status
	attempt.FailureReason = chargeResult.ErrorReason
	attempt.FailureCode = chargeResult.ErrorCode
	attempt.CompletedAt = chargeResult.ProcessedAt
	attempt.ProcessorResponse = chargeResult.RawData

	// Save attempt
	attempt, err = s.dunningRepository.CreateAttempt(ctx, attempt)
	if err != nil {
		s.logger.Error("Failed to create dunning attempt", err.Error())
		return dunning.DunningAttempt{}, err
	}

	// Publish event for the attempt result
	// For manual attempts, we don't suspend the subscription or send a final notice
	event := topic.NewDunningAttemptEvent(attempt, campaign, false, false)

	eventTopic := topic.DunningAttemptFailed
	if chargeResult.Status == payments.PaymentStatusSucceeded {
		eventTopic = topic.DunningAttemptSucceeded
	}

	err = s.pubsub.Publish(attempt.OrgId, eventTopic, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning attempt event", err.Error())
		// Continue anyway, as the attempt has already been created
	}

	return attempt, nil
}

// HandleChargeResult creates a Attempt and updates the Dunning Campaign based on the charge result
func (s *DunningService) HandleChargeResult(
	ctx context.Context,
	campaign dunning.DunningCampaign,
	chargeResult payments.ChargeResult,
	config dunning.DunningConfig,
) (dto.HandleChargeResultResponse, error) {
	var zero = dto.HandleChargeResultResponse{}
	// Find campaign
	campaign, err := s.dunningRepository.FindCampaignById(ctx, campaign.OrgId, campaign.Id)
	if err != nil {
		s.logger.Error("Failed to find dunning campaign", err.Error())
		return zero, err
	}

	// Validate campaign status
	if campaign.Status != dunning.DunningStatusActive {
		s.logger.Info("Campaign is not active", "status", campaign.Status)
		return zero, lib.NewCustomError(lib.BadRequestError, "campaign is not active", nil)
	}

	// Get the subscription
	subscription, err := s.subscriptionRepository.FindById(ctx, campaign.OrgId, campaign.SubscriptionId)
	if err != nil {
		s.logger.Error("Failed to find subscription", err.Error())
		return zero, err
	}

	// Create attempt
	attempt := dunning.DunningAttempt{
		OrgId:             campaign.OrgId,
		Id:                lib.GenerateId("dun"),
		DunningCampaignId: campaign.Id,
		SubscriptionId:    campaign.SubscriptionId,
		AttemptNumber:     campaign.TotalAttempts + 1,
		AttemptType:       dunning.DunningAttemptTypeProgressive,
		Amount:            chargeResult.Amount,
		Currency:          campaign.Currency,
		PaymentMethodId:   subscription.PaymentMethodId,
		Status:            chargeResult.Status,
		FailureReason:     chargeResult.ErrorReason,
		FailureCode:       chargeResult.ErrorCode,
		ProcessorResponse: chargeResult.RawData,
		ProcessingTimeMs:  0,
		AttemptedAt:       time.Now().UTC(),
		CompletedAt:       chargeResult.ProcessedAt,
		TriggeredBy:       string(dunning.DunningAttemptTypeProgressive),
		Metadata:          nil,
		CreatedAt:         time.Now().UTC(),
	}
	// Save attempt
	attempt, err = s.dunningRepository.CreateAttempt(ctx, attempt)
	if err != nil {
		s.logger.Error("Failed to create dunning attempt", "err", err.Error())
		return zero, err
	}

	// Update campaign statistics
	campaign.LastAttemptAt = attempt.AttemptedAt
	campaign.TotalAttempts++
	if attempt.AttemptType == dunning.DunningAttemptTypeProgressive {
		campaign.ProgressiveAttempts++
	}
	campaign.UpdatedAt = time.Now().UTC()

	// Handle successful payment
	if attempt.Status == payments.PaymentStatusSucceeded {
		s.logger.Info("Payment attempt succeeded, handling recovery")

		// Mark campaign as recovered
		campaign.Status = dunning.DunningStatusRecovered
		campaign.RecoveryMethod = string(attempt.AttemptType)
		campaign.RecoveredAmount = campaign.FailedAmount
		campaign.RecoveredAt = time.Now().UTC()
		campaign.CompletedAt = time.Now().UTC()

		// Update the campaign in the database
		updatedCampaign, err := s.dunningRepository.UpdateCampaign(ctx, campaign)
		if err != nil {
			s.logger.Error("Failed to update campaign", "Error", err.Error())
			return zero, err
		}

		// Publish recovery event
		event := topic.NewDunningCampaignEvent(updatedCampaign)
		err = s.pubsub.Publish(updatedCampaign.OrgId, topic.DunningCampaignRecovered, event)
		if err != nil {
			s.logger.Error("Failed to publish dunning campaign recovered event", "Error", err.Error())
		}

		return dto.HandleChargeResultResponse{
			Subscription: subscription,
			Campaign:     updatedCampaign,
			Attempt:      attempt,
		}, nil
	}

	// Handle failed payment - check escalation rules
	s.logger.Info("Payment attempt failed, checking escalation rules",
		"AttemptNumber", attempt.AttemptNumber,
		"FailureReason", attempt.FailureReason,
		"FailureCode", attempt.FailureCode)

	// Check if we need to cancel subscription (final failure)
	shouldCancel := attempt.AttemptNumber >= config.EscalationRules.CancelAfterAttempt &&
		attempt.AttemptType == dunning.DunningAttemptTypeProgressive

	if shouldCancel {
		s.logger.Info("Cancelling subscription due to escalation rules", "AttemptNumber", attempt.AttemptNumber)
		// Mark campaign as failed
		campaign.Status = dunning.DunningStatusFailed
		campaign.FinalFailureReason = "max_attempts_reached"
		campaign.CompletedAt = time.Now().UTC()

		// Update the campaign in the database
		updatedCampaign, err := s.UpdateCampaign(ctx, attempt.OrgId, campaign)
		if err != nil {
			s.logger.Error("Failed to update campaign", "Error", err.Error())
			return zero, err
		}

		subscription, err = s.subscriptionService.CancelSubscription(ctx, subscriptions.CancelSubscriptionInput{
			OrgId:  campaign.OrgId,
			Id:     campaign.SubscriptionId,
			Reason: "Max dunning attempts reached",
		})
		if err != nil {
			s.logger.Error("Failed to cancel subscription", "Error", err.Error())
			return zero, err
		}

		// Publish failed event
		event := topic.NewDunningCampaignEvent(updatedCampaign)
		err = s.pubsub.Publish(updatedCampaign.OrgId, topic.DunningCampaignFailed, event)
		if err != nil {
			s.logger.Error("Failed to publish dunning campaign failed event", "Error", err.Error())
		}

		return dto.HandleChargeResultResponse{
			Subscription: subscription,
			Campaign:     updatedCampaign,
			Attempt:      attempt,
		}, nil
	}

	// For regular failures, just update the campaign
	updatedCampaign, err := s.UpdateCampaign(ctx, attempt.OrgId, campaign)
	if err != nil {
		s.logger.Error("Failed to update campaign", "Error", err.Error())
		return zero, err
	}

	// Publish attempt failed event
	event := topic.NewDunningAttemptEvent(attempt, updatedCampaign, false, shouldCancel)
	err = s.pubsub.Publish(attempt.OrgId, topic.DunningAttemptFailed, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning attempt failed event", "Error", err.Error())
	}

	// Publish event for the attempt result
	// For manual attempts, we don't suspend the subscription or send a final notice
	event = topic.NewDunningAttemptEvent(attempt, campaign, false, false)

	eventTopic := topic.DunningAttemptFailed
	if chargeResult.Status == payments.PaymentStatusSucceeded {
		eventTopic = topic.DunningAttemptSucceeded
	}

	err = s.pubsub.Publish(attempt.OrgId, eventTopic, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning attempt event", err.Error())
		// Continue anyway, as the attempt has already been created
	}

	return dto.HandleChargeResultResponse{
		Subscription: subscription,
		Campaign:     updatedCampaign,
		Attempt:      attempt,
	}, nil
}

// ListCommunicationsByCampaign lists dunning communications by campaign ID
func (s *DunningService) ListCommunicationsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningCommunication, int, error) {
	s.logger.Info("Listing dunning communications by campaign", "orgId", orgId, "campaignId", campaignId)

	communications, total, err := s.dunningRepository.FindCommunicationsByCampaignId(ctx, orgId, campaignId, entities.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortDirection: pagination.SortDirection,
		SortBy:        pagination.SortBy,
	})
	if err != nil {
		s.logger.Error("Failed to list dunning communications by campaign", err.Error())
		return nil, 0, err
	}

	return communications, total, nil
}

// CreatePaymentUpdateToken creates a payment update token
func (s *DunningService) CreatePaymentUpdateToken(ctx context.Context, input dto.CreatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error) {
	s.logger.Info("Creating payment update token", "orgId", input.OrgId, "subscriptionId", input.SubscriptionId)

	// Validate subscription exists
	_, err := s.subscriptionRepository.FindById(ctx, input.OrgId, input.SubscriptionId)
	if err != nil {
		s.logger.Error("Failed to find subscription", err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	// Validate customer exists
	_, err = s.customerRepository.FindById(ctx, input.OrgId, input.CustomerId)
	if err != nil {
		s.logger.Error("Failed to find customer", err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	// Set default values
	maxUses := 5
	if input.MaxUses > 0 {
		maxUses = input.MaxUses
	}

	expiryHours := 72
	if input.ExpiryHours > 0 {
		expiryHours = input.ExpiryHours
	}

	allowedActions := map[string]bool{
		"update_payment_method": true,
		"retry_payment":         true,
	}
	if input.AllowedActions != nil {
		allowedActions = input.AllowedActions
	}

	// Create token data
	tokenData := map[string]interface{}{
		"subscription_id": input.SubscriptionId,
		"customer_id":     input.CustomerId,
		"org_id":          input.OrgId,
		"expires_at":      time.Now().UTC().Add(time.Hour * time.Duration(expiryHours)),
		"max_uses":        maxUses,
		"allowed_actions": allowedActions,
	}

	// Create token
	token := dunning.PaymentUpdateToken{
		OrgId:             input.OrgId,
		TokenId:           lib.GenerateId("tok"),
		SubscriptionId:    input.SubscriptionId,
		CustomerId:        input.CustomerId,
		DunningCampaignId: input.DunningCampaignId,
		TokenData:         tokenData,
		Signature:         "", // This would be set by the repository
		ExpiresAt:         time.Now().UTC().Add(time.Hour * time.Duration(expiryHours)),
		MaxUses:           maxUses,
		UsedCount:         0,
		Status:            dunning.TokenStatusActive,
		AllowedActions:    allowedActions,
		AdminGenerated:    input.AdminGenerated,
		AdminUserId:       input.AdminUserId,
		AdminReason:       input.AdminReason,
		AdminNotes:        input.AdminNotes,
		CreatedBy:         input.CreatedBy,
		CreatedAt:         time.Now().UTC(),
	}

	// Save token
	token, err = s.dunningRepository.CreateToken(ctx, token)
	if err != nil {
		s.logger.Error("Failed to create payment update token", err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	// Publish event
	event := topic.NewDunningTokenEvent(token)
	err = s.pubsub.Publish(token.OrgId, topic.DunningTokenCreated, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning token created event", err.Error())
		// Continue anyway, as the token has already been created
	}

	return token, nil
}

// VerifyPaymentUpdateToken verifies a payment update token
func (s *DunningService) VerifyPaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error) {
	s.logger.Info("Verifying payment update token", "orgId", orgId, "tokenId", tokenId)

	// Find token
	token, err := s.dunningRepository.FindTokenById(ctx, orgId, tokenId)
	if err != nil {
		s.logger.Error("Failed to find payment update token", err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	// Validate token status
	if token.Status != dunning.TokenStatusActive {
		s.logger.Info("Token is not active", "status", token.Status)
		return dunning.PaymentUpdateToken{}, lib.NewCustomError(lib.BadRequestError, "token is not active", nil)
	}

	// Validate token expiry
	if token.ExpiresAt.Before(time.Now().UTC()) {
		s.logger.Info("Token has expired", "expiresAt", token.ExpiresAt)
		token.Status = dunning.TokenStatusExpired
		_, err = s.dunningRepository.UpdateToken(ctx, token)
		if err != nil {
			s.logger.Error("Failed to update token status", err.Error())
			// Continue anyway, as we're returning an error
		}
		return dunning.PaymentUpdateToken{}, lib.NewCustomError(lib.BadRequestError, "token has expired", nil)
	}

	// Validate token usage
	if token.UsedCount >= token.MaxUses {
		s.logger.Info("Token has reached maximum usage", "usedCount", token.UsedCount, "maxUses", token.MaxUses)
		token.Status = dunning.TokenStatusMaxUsesReached
		_, err = s.dunningRepository.UpdateToken(ctx, token)
		if err != nil {
			s.logger.Error("Failed to update token status", err.Error())
			// Continue anyway, as we're returning an error
		}
		return dunning.PaymentUpdateToken{}, lib.NewCustomError(lib.BadRequestError, "token has reached maximum usage", nil)
	}

	return token, nil
}

// ActivatePaymentUpdateToken activates a payment update token
func (s *DunningService) ActivatePaymentUpdateToken(ctx context.Context, input dto.ActivatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error) {
	s.logger.Info("Activating payment update token", "orgId", input.OrgId, "tokenId", input.TokenId)

	// Verify token
	token, err := s.VerifyPaymentUpdateToken(ctx, input.OrgId, input.TokenId)
	if err != nil {
		return dunning.PaymentUpdateToken{}, err
	}

	// Update token
	token.UsedCount++
	token.LastUsedAt = time.Now().UTC()
	token.LastUsedIp = input.IpAddress

	// Check if token has reached maximum usage
	if token.UsedCount >= token.MaxUses {
		token.Status = dunning.TokenStatusMaxUsesReached
	}

	// Save token
	token, err = s.dunningRepository.UpdateToken(ctx, token)
	if err != nil {
		s.logger.Error("Failed to update payment update token", err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	// Publish event
	event := topic.NewDunningTokenEvent(token)
	err = s.pubsub.Publish(token.OrgId, topic.DunningTokenActivated, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning token activated event", err.Error())
		// Continue anyway, as the token has already been updated
	}

	return token, nil
}

// RevokePaymentUpdateToken revokes a payment update token
func (s *DunningService) RevokePaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error) {
	s.logger.Info("Revoking payment update token", "orgId", orgId, "tokenId", tokenId)

	// Find token
	token, err := s.dunningRepository.FindTokenById(ctx, orgId, tokenId)
	if err != nil {
		s.logger.Error("Failed to find payment update token", err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	// Validate token status
	if token.Status != dunning.TokenStatusActive {
		s.logger.Info("Token is not active", "status", token.Status)
		return dunning.PaymentUpdateToken{}, lib.NewCustomError(lib.BadRequestError, "token is not active", nil)
	}

	// Update token
	token.Status = dunning.TokenStatusRevoked

	// Save token
	token, err = s.dunningRepository.UpdateToken(ctx, token)
	if err != nil {
		s.logger.Error("Failed to update payment update token", err.Error())
		return dunning.PaymentUpdateToken{}, err
	}

	// Publish event
	event := topic.NewDunningTokenEvent(token)
	err = s.pubsub.Publish(token.OrgId, topic.DunningTokenRevoked, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning token revoked event", err.Error())
		// Continue anyway, as the token has already been updated
	}

	return token, nil
}

// CreateConfiguration creates a dunning configuration
func (s *DunningService) CreateConfiguration(ctx context.Context, input dto.CreateDunningConfigurationInput) (dunning.DunningConfiguration, error) {
	s.logger.Info("Creating dunning configuration", "orgId", input.OrgId, "name", input.Name)

	// Convert DunningConfig to map[string]interface{} using JSON marshaling
	configBytes, err := json.Marshal(input.Config)
	if err != nil {
		s.logger.Error("Failed to marshal dunning configuration", err.Error())
		return dunning.DunningConfiguration{}, err
	}

	var configMap map[string]interface{}
	err = json.Unmarshal(configBytes, &configMap)
	if err != nil {
		s.logger.Error("Failed to unmarshal dunning configuration", err.Error())
		return dunning.DunningConfiguration{}, err
	}

	// Create configuration
	config := dunning.DunningConfiguration{
		OrgId:            input.OrgId,
		Id:               lib.GenerateId("dcfg"),
		Name:             input.Name,
		Description:      input.Description,
		Priority:         input.Priority,
		AppliesTo:        input.AppliesTo,
		TargetRules:      input.TargetRules,
		Config:           configMap,
		Status:           dunning.ConfigStatusActive,
		IsAbTest:         input.IsAbTest,
		AbTestPercentage: input.AbTestPercentage,
		CreatedBy:        input.CreatedBy,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	// Save configuration
	config, err = s.dunningRepository.CreateConfiguration(ctx, config)
	if err != nil {
		s.logger.Error("Failed to create dunning configuration", err.Error())
		return dunning.DunningConfiguration{}, err
	}

	// Publish event
	event := topic.NewDunningConfigurationEvent(config)
	err = s.pubsub.Publish(config.OrgId, topic.DunningConfigurationCreated, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning configuration created event", err.Error())
		// Continue anyway, as the configuration has already been created
	}

	return config, nil
}

// GetConfiguration gets a dunning configuration by ID
func (s *DunningService) GetConfiguration(ctx context.Context, orgId string, id string) (dunning.DunningConfiguration, error) {
	s.logger.Info("Getting dunning configuration", "orgId", orgId, "id", id)

	config, err := s.dunningRepository.FindConfigurationById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find dunning configuration", err.Error())
		return dunning.DunningConfiguration{}, err
	}

	return config, nil
}

// ListConfigurations lists dunning configurations
func (s *DunningService) ListConfigurations(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningConfiguration, int, error) {
	s.logger.Info("Listing dunning configurations", "orgId", orgId)

	configs, total, err := s.dunningRepository.FindConfigurations(ctx, orgId, entities.Pagination{
		Page:          pagination.Page,
		Limit:         pagination.Limit,
		Offset:        pagination.Offset,
		SortDirection: pagination.SortDirection,
		SortBy:        pagination.SortBy,
	})
	if err != nil {
		s.logger.Error("Failed to list dunning configurations", err.Error())
		return nil, 0, err
	}

	return configs, total, nil
}

// UpdateConfiguration updates a dunning configuration
func (s *DunningService) UpdateConfiguration(ctx context.Context, input dto.UpdateDunningConfigurationInput) (dunning.DunningConfiguration, error) {
	s.logger.Info("Updating dunning configuration", "orgId", input.OrgId, "id", input.Id)

	// Find configuration
	config, err := s.dunningRepository.FindConfigurationById(ctx, input.OrgId, input.Id)
	if err != nil {
		s.logger.Error("Failed to find dunning configuration", err.Error())
		return dunning.DunningConfiguration{}, err
	}

	// Update configuration
	if input.Name != "" {
		config.Name = input.Name
	}
	if input.Description != "" {
		config.Description = input.Description
	}
	if input.Priority != 0 {
		config.Priority = input.Priority
	}
	if input.AppliesTo != "" {
		config.AppliesTo = input.AppliesTo
	}
	if input.TargetRules != nil {
		config.TargetRules = input.TargetRules
	}
	// Convert DunningConfig to map[string]interface{} using JSON marshaling
	configBytes, err := json.Marshal(input.Config)
	if err != nil {
		s.logger.Error("Failed to marshal dunning configuration", err.Error())
		return dunning.DunningConfiguration{}, err
	}

	var configMap map[string]interface{}
	err = json.Unmarshal(configBytes, &configMap)
	if err != nil {
		s.logger.Error("Failed to unmarshal dunning configuration", err.Error())
		return dunning.DunningConfiguration{}, err
	}

	config.Config = configMap
	if input.Status != "" {
		config.Status = input.Status
	}
	if input.IsAbTest {
		config.IsAbTest = input.IsAbTest
		config.AbTestPercentage = input.AbTestPercentage
	}
	config.UpdatedAt = time.Now().UTC()

	// Save configuration
	config, err = s.dunningRepository.UpdateConfiguration(ctx, config)
	if err != nil {
		s.logger.Error("Failed to update dunning configuration", err.Error())
		return dunning.DunningConfiguration{}, err
	}

	// Publish event
	event := topic.NewDunningConfigurationEvent(config)
	err = s.pubsub.Publish(config.OrgId, topic.DunningConfigurationUpdated, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning configuration updated event", err.Error())
		// Continue anyway, as the configuration has already been updated
	}

	return config, nil
}

// GetCustomerDunningHistory gets a customer's dunning history
func (s *DunningService) GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (dunning.CustomerDunningHistory, error) {
	s.logger.Info("Getting customer dunning history", "orgId", orgId, "customerId", customerId)

	history, err := s.dunningRepository.GetCustomerDunningHistory(ctx, orgId, customerId)
	if err != nil {
		// If the history doesn't exist, create a new one
		var customErr lib.CustomError
		if errors.As(err, &customErr) && customErr.Type == lib.NotFoundError {
			// Validate customer exists
			_, err := s.customerRepository.FindById(ctx, orgId, customerId)
			if err != nil {
				s.logger.Error("Failed to find customer", err.Error())
				return dunning.CustomerDunningHistory{}, err
			}

			// Create new history
			history = dunning.CustomerDunningHistory{
				OrgId:                 orgId,
				CustomerId:            customerId,
				TotalDunningCampaigns: 0,
				SuccessfulRecoveries:  0,
				FailedCampaigns:       0,
				TotalAmountAtRisk:     0,
				TotalAmountRecovered:  0,
				TotalAmountLost:       0,
				UpdatedAt:             time.Now().UTC(),
			}

			// Save history
			history, err = s.dunningRepository.UpdateCustomerDunningHistory(ctx, history)
			if err != nil {
				s.logger.Error("Failed to create customer dunning history", err.Error())
				return dunning.CustomerDunningHistory{}, err
			}

			return history, nil
		}

		s.logger.Error("Failed to get customer dunning history", err.Error())
		return dunning.CustomerDunningHistory{}, err
	}

	return history, nil
}
