package services

import (
	"context"
	"encoding/json"
	"payloop/internal/lib"
	"time"

	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/repositories"
)

// DunningOrchestrationService is an extension of the DunningService that orchestrates dunning workflows.
type DunningOrchestrationService struct {
	interfaces.DunningService
	subscriptionService    interfaces.SubscriptionService
	subscriptionRepository repositories.SubscriptionRepository
	workflowEngine         interfaces.Engine
	pubsub                 events.NotificationPublisher
	errorReporter          lib.ErrorReporter
	logger                 logger.Logger
}

// NewDunningOrchestrationService creates a new DunningOrchestrationService
func NewDunningOrchestrationService(
	dunningService interfaces.DunningService,
	subscriptionService interfaces.SubscriptionService,
	subscriptionRepository repositories.SubscriptionRepository,
	workflowEngine interfaces.Engine,
	pubsub events.NotificationPublisher,
	errorReporter lib.ErrorReporter,
	logger logger.Logger,
) interfaces.DunningOrchestrationService {
	svc := &DunningOrchestrationService{
		DunningService:         dunningService,
		subscriptionService:    subscriptionService,
		subscriptionRepository: subscriptionRepository,
		workflowEngine:         workflowEngine,
		errorReporter:          errorReporter,
		pubsub:                 pubsub,
		logger:                 logger,
	}

	logger.Debugf("[DunningOrchestrationService] Subscribing to all topics")
	_, err := pubsub.Subscribe(topic.SubscriptionPaymentChargeFailed, svc.HandleSubscriptionPastDue)
	if err != nil {
		logger.Error("Failed to subscribe to topic", err.Error())
		panic(err)
	}

	return svc
}

// HandleSubscriptionPastDue starts a dunning workflow when a subscription payment fails
func (s DunningOrchestrationService) HandleSubscriptionPastDue(t string, data []byte) {
	s.logger.Infof("[DunningOrchestrationService] checking topic: %s", t)
	// Check if the org is subscribed to any outgoing messages and send them using a workflow

	var payload events.Payload
	err := json.Unmarshal(data, &payload)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal payload: %v", err)
		return
	}
	var chargeFailureEvent topic.SubscriptionPaymentChargeFailureEvent
	payloadBytes, err := json.Marshal(payload.Data)
	if err != nil {
		s.logger.Errorf("Failed to marshal payload data: %v", err)
		return
	}
	err = json.Unmarshal(payloadBytes, &chargeFailureEvent)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal event data: %v", err)
		return
	}

	subscription := chargeFailureEvent.Subscription
	chargeResult := chargeFailureEvent.ChargeResult

	_, _, err = s.workflowEngine.StartDunningWorkflow(context.Background(), dto.StartDunningWorkflowInput{
		OrgId:                subscription.OrgId,
		SubscriptionId:       subscription.Id,
		CustomerId:           subscription.CustomerId,
		FailedAmount:         int(chargeResult.Amount),
		Currency:             chargeResult.Currency,
		InitialFailureReason: chargeResult.ErrorReason,
		ParentWorkflowId:     "", // This will be set by the dunning service if needed
		PaymentResult:        chargeResult,
		Metadata: map[string]string{
			"triggered_by":    "subscription_charge_failure",
			"failure_code":    chargeResult.ErrorCode,
			"subscription_id": subscription.Id,
		},
	})

	if err != nil {
		s.logger.Errorf("Failed to start workflow %v", err.Error())
		s.errorReporter.ReportError(context.Background(), err, map[string]interface{}{
			"operation":       "start_dunning_workflow",
			"org_id":          subscription.OrgId,
			"subscription_id": subscription.Id,
			"failed_amount":   chargeResult.Amount,
			"failure_reason":  chargeResult.ErrorReason,
		})
	}
}

// StartDunningWorkflow starts a dunning workflow for a failed payment
func (s *DunningOrchestrationService) StartDunningWorkflow(ctx context.Context, input dto.StartDunningWorkflowInput) (dunning.DunningCampaign, error) {
	s.logger.Info("Starting dunning workflow", "OrgId", input.OrgId, "SubscriptionId", input.SubscriptionId)

	// Create a campaign record
	campaign, err := s.CreateCampaign(ctx, dto.CreateDunningCampaignInput{
		OrgId:                input.OrgId,
		SubscriptionId:       input.SubscriptionId,
		CustomerId:           input.CustomerId,
		FailedAmount:         input.FailedAmount,
		Currency:             input.Currency,
		InitialFailureReason: input.InitialFailureReason,
		ParentWorkflowId:     input.ParentWorkflowId,
		Metadata:             input.Metadata,
	})
	if err != nil {
		s.logger.Error("Failed to create dunning campaign", "err", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Start the workflow using the workflow engine
	workflowId, runId, err := s.workflowEngine.StartDunningWorkflow(ctx, input)
	if err != nil {
		s.logger.Error("Failed to start dunning workflow", "err", err.Error())
		return campaign, err
	}

	// Update the campaign with the workflow ID and run ID
	campaign.TemporalWorkflowId = workflowId
	campaign.TemporalRunId = runId

	// Update the subscription status
	subscription, err := s.subscriptionService.FindById(ctx, input.OrgId, input.SubscriptionId)
	if err != nil {
		s.logger.Error("Failed to find subscription", "err", err.Error())
	} else {
		// Update subscription metadata
		subscription.Metadata["dunning_status"] = string(dunning.DunningStatusActive)
		subscription.Metadata["dunning_started_at"] = time.Now().UTC().Format(time.RFC3339)
		subscription.Metadata["dunning_campaign_id"] = campaign.Id

		// Update subscription
		_, err = s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to update subscription", "err", err.Error())
		}
	}

	// Publish event
	event := topic.NewDunningCampaignEvent(campaign)
	err = s.pubsub.Publish(campaign.OrgId, topic.DunningCampaignStarted, event)
	if err != nil {
		s.logger.Error("Failed to publish dunning campaign started event", "err", err.Error())
	}

	return campaign, nil
}

// HandlePaymentMethodUpdated handles a payment method update
func (s *DunningOrchestrationService) HandlePaymentMethodUpdated(ctx context.Context, input dto.PaymentMethodUpdatedInput) error {
	s.logger.Info("Handling payment method update", "OrgId", input.OrgId, "SubscriptionId", input.SubscriptionId)

	// If no campaign ID is provided, find the active campaign for the subscription
	campaignId := input.DunningCampaignId
	if campaignId == "" {
		campaigns, _, err := s.ListCampaignsBySubscription(ctx, input.OrgId, input.SubscriptionId, request.Pagination{
			Page:          1,
			Limit:         10,
			SortDirection: "desc",
			SortBy:        "created_at",
		})
		if err != nil {
			s.logger.Error("Failed to list dunning campaigns", "err", err.Error())
			return err
		}

		// Find the active campaign
		for _, campaign := range campaigns {
			if campaign.Status == dunning.DunningStatusActive {
				campaignId = campaign.Id
				break
			}
		}

		if campaignId == "" {
			s.logger.Info("No active dunning campaign found for subscription", "SubscriptionId", input.SubscriptionId)
			return nil
		}
	}

	// Get the campaign to get the workflow ID
	campaign, err := s.FindCampaignById(ctx, input.OrgId, campaignId)
	if err != nil {
		s.logger.Error("Failed to find dunning campaign", "err", err.Error())
		return err
	}

	// Check if the campaign has a workflow ID
	if campaign.TemporalWorkflowId == "" {
		s.logger.Error("Campaign has no workflow ID", "CampaignId", campaignId)
		return nil
	}

	// Signal the workflow to trigger an immediate retry
	err = s.workflowEngine.SignalDunningWorkflow(ctx, campaign.TemporalWorkflowId, "payment_method.updated", input)
	if err != nil {
		s.logger.Error("Failed to signal dunning workflow", "err", err.Error())
		return err
	}

	return nil
}

// HandleSubscriptionStateChanged handles a subscription state change
func (s *DunningOrchestrationService) HandleSubscriptionStateChanged(ctx context.Context, input dto.SubscriptionStateChangedInput) error {
	s.logger.Info("Handling subscription state change", "OrgId", input.OrgId, "SubscriptionId", input.SubscriptionId)

	// If no campaign ID is provided, find the active campaign for the subscription
	campaignId := input.DunningCampaignId
	if campaignId == "" {
		campaigns, _, err := s.ListCampaignsBySubscription(ctx, input.OrgId, input.SubscriptionId, request.Pagination{
			Page:          1,
			Limit:         10,
			SortDirection: "desc",
			SortBy:        "created_at",
		})
		if err != nil {
			s.logger.Error("Failed to list dunning campaigns", "err", err.Error())
			return err
		}

		// Find the active campaign
		for _, campaign := range campaigns {
			if campaign.Status == dunning.DunningStatusActive || campaign.Status == dunning.DunningStatusPaused {
				campaignId = campaign.Id
				break
			}
		}

		if campaignId == "" {
			s.logger.Info("No active dunning campaign found for subscription", "SubscriptionId", input.SubscriptionId)
			return nil
		}
	}

	// Get the campaign to get the workflow ID
	campaign, err := s.FindCampaignById(ctx, input.OrgId, campaignId)
	if err != nil {
		s.logger.Error("Failed to find dunning campaign", "err", err.Error())
		return err
	}

	// Check if the campaign has a workflow ID
	if campaign.TemporalWorkflowId == "" {
		s.logger.Error("Campaign has no workflow ID", "CampaignId", campaignId)
		return nil
	}

	// Signal the workflow with the subscription state change
	err = s.workflowEngine.SignalDunningWorkflow(ctx, campaign.TemporalWorkflowId, "subscription.state_changed", input)
	if err != nil {
		s.logger.Error("Failed to signal dunning workflow", "err", err.Error())
		return err
	}

	return nil
}

// HandleDunningAttemptResult handles a dunning attempt result
func (s *DunningOrchestrationService) HandleDunningAttemptResult(ctx context.Context, input dto.DunningAttemptResultInput) (dunning.DunningCampaign, error) {
	s.logger.Info("Handling dunning attempt result", "OrgId", input.OrgId, "CampaignId", input.CampaignId)

	// Get the campaign
	campaign, err := s.FindCampaignById(ctx, input.OrgId, input.CampaignId)
	if err != nil {
		s.logger.Error("Failed to find dunning campaign", "err", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Update the campaign based on the attempt result
	if input.Success {
		// Mark the campaign as recovered
		campaign.Status = dunning.DunningStatusRecovered
		campaign.RecoveryMethod = "payment_retry"
		campaign.RecoveredAmount = campaign.FailedAmount
		campaign.RecoveredAt = time.Now().UTC()
		campaign.CompletedAt = time.Now().UTC()

		// Update the subscription
		subscription, err := s.subscriptionService.FindById(ctx, input.OrgId, campaign.SubscriptionId)
		if err != nil {
			s.logger.Error("Failed to find subscription", "err", err.Error())
		} else {
			// Update subscription metadata
			subscription.Metadata["dunning_status"] = ""
			subscription.Metadata["dunning_completed_at"] = time.Now().UTC().Format(time.RFC3339)
			subscription.Metadata["last_dunning_recovery_at"] = time.Now().UTC().Format(time.RFC3339)

			// Update subscription
			subscription.Status = entities.SubscriptionStatusActive
			_, err = s.subscriptionRepository.Update(ctx, subscription)
			if err != nil {
				s.logger.Error("Failed to update subscription", "err", err.Error())
			}
		}

		// Publish event
		event := topic.NewDunningCampaignEvent(campaign)
		err = s.pubsub.Publish(campaign.OrgId, topic.DunningCampaignRecovered, event)
		if err != nil {
			s.logger.Error("Failed to publish dunning campaign recovered event", "err", err.Error())
		}
	} else {
		// Update the campaign with the attempt information
		campaign.LastAttemptAt = time.Now().UTC()
		campaign.TotalAttempts++

		// Create a simplified DunningAttempt object for the event
		attempt := dunning.DunningAttempt{
			OrgId:             campaign.OrgId,
			Id:                input.AttemptId,
			DunningCampaignId: campaign.Id,
			SubscriptionId:    campaign.SubscriptionId,
			Status:            input.PaymentResult.Status,
			FailureReason:     input.PaymentResult.ErrorReason,
			FailureCode:       input.PaymentResult.ErrorCode,
			AttemptedAt:       time.Now().UTC(),
		}

		// Publish event
		event := topic.NewDunningAttemptEvent(attempt, campaign, false, false)
		err = s.pubsub.Publish(campaign.OrgId, topic.DunningAttemptFailed, event)
		if err != nil {
			s.logger.Error("Failed to publish dunning attempt failed event", "err", err.Error())
		}
	}

	// Save the updated campaign
	updatedCampaign, err := s.UpdateCampaign(ctx, input.OrgId, campaign)
	if err != nil {
		s.logger.Error("Failed to update dunning campaign", "err", err.Error())
		return campaign, err
	}

	return updatedCampaign, nil
}
