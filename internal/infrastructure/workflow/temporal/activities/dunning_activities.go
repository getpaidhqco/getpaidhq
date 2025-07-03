package activities

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"

	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/lib"
)

// AttemptContext provides context about the current attempt within the dunning campaign
type AttemptContext struct {
	AttemptNumber            int  `json:"attempt_number"`
	WasSubscriptionSuspended bool `json:"was_subscription_suspended"`
}

// DunningActivities contains activities for the DunningWorkflow
type DunningActivities struct {
	dunningService      interfaces.DunningService
	subscriptionService interfaces.SubscriptionService
	pubsub              events.NotificationPublisher
	errorReporter       lib.ErrorReporter
}

// NewDunningActivities creates a new DunningActivities
func NewDunningActivities(
	dunningService interfaces.DunningService,
	subscriptionService interfaces.SubscriptionService,
	pubsub events.NotificationPublisher,
	errorReporter lib.ErrorReporter,
) DunningActivities {
	return DunningActivities{
		dunningService:      dunningService,
		subscriptionService: subscriptionService,
		pubsub:              pubsub,
		errorReporter:       errorReporter,
	}
}

// DunningWorkflowInput represents the input for the DunningWorkflow
type DunningWorkflowInput struct {
	OrgId                string                `json:"org_id"`
	SubscriptionId       string                `json:"subscription_id"`
	CustomerId           string                `json:"customer_id"`
	FailedAmount         int                   `json:"failed_amount"`
	Currency             string                `json:"currency"`
	InitialFailureReason string                `json:"initial_failure_reason,omitempty"`
	ParentWorkflowId     string                `json:"parent_workflow_id,omitempty"`
	PaymentResult        payments.ChargeResult `json:"payment_result"`
	Metadata             map[string]string     `json:"metadata,omitempty"`
}

// CreateDunningCampaign creates a new dunning campaign
func (a *DunningActivities) CreateDunningCampaign(ctx context.Context, input DunningWorkflowInput) (dunning.DunningCampaign, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating dunning campaign",
		"OrgId", input.OrgId,
		"SubscriptionId", input.SubscriptionId)

	campaign, err := a.dunningService.CreateCampaign(ctx, interfaces.CreateDunningCampaignInput{
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
		logger.Error("Failed to create dunning campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	return campaign, nil
}

// ResolveDunningConfig resolves the dunning configuration for an organization
func (a *DunningActivities) ResolveDunningConfig(ctx context.Context, orgId string) (dunning.DunningConfig, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Resolving dunning configuration", "OrgId", orgId)

	// Get all configurations for the organization
	configs, _, err := a.dunningService.ListConfigurations(ctx, orgId, request.Pagination{
		Page:          1,
		Limit:         100,
		SortDirection: "desc",
		SortBy:        "priority",
	})
	if err != nil {
		logger.Error("Failed to list dunning configurations", "Error", err.Error())
		return dunning.DefaultDunningConfig(), err
	}

	// If no configurations found, return default
	if len(configs) == 0 {
		logger.Info("No dunning configurations found, using default")
		return dunning.DefaultDunningConfig(), nil
	}

	// Find the highest priority active configuration
	highestPriority := -1

	for _, config := range configs {
		if config.Status == dunning.ConfigStatusActive && config.Priority > highestPriority {
			highestPriority = config.Priority
		}
	}

	// If no active configuration found, return default
	if highestPriority == -1 {
		logger.Info("No active dunning configurations found, using default")
		return dunning.DefaultDunningConfig(), nil
	}

	// Extract the DunningConfig from the configuration
	// Since Config is stored as a map[string]interface{}, we need to convert it to a DunningConfig
	// In a real implementation, this would be handled by the service layer
	// For now, we'll just return the default config
	logger.Info("Using default dunning configuration")
	return dunning.DefaultDunningConfig(), nil
}

// TriggerImmediateRetry triggers an immediate retry for a dunning campaign
func (a *DunningActivities) TriggerImmediateRetry(ctx context.Context, orgId string, campaignId string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Triggering immediate retry", "OrgId", orgId, "CampaignId", campaignId)

	// Trigger a manual attempt
	_, err := a.dunningService.TriggerManualAttempt(ctx, interfaces.TriggerManualAttemptInput{
		OrgId:       orgId,
		CampaignId:  campaignId,
		TriggeredBy: "payment_method_updated",
	})
	if err != nil {
		logger.Error("Failed to trigger manual attempt", "Error", err.Error())
		return err
	}

	return nil
}

// PauseDunningCampaign pauses a dunning campaign
func (a *DunningActivities) PauseDunningCampaign(ctx context.Context, orgId string, campaignId string) (dunning.DunningCampaign, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Pausing dunning campaign", "OrgId", orgId, "CampaignId", campaignId)

	campaign, err := a.dunningService.PauseCampaign(ctx, interfaces.PauseDunningCampaignInput{
		OrgId:  orgId,
		Id:     campaignId,
		Reason: "manual_pause",
	})
	if err != nil {
		logger.Error("Failed to pause dunning campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	return campaign, nil
}

// ResumeDunningCampaign resumes a paused dunning campaign
func (a *DunningActivities) ResumeDunningCampaign(ctx context.Context, orgId string, campaignId string) (dunning.DunningCampaign, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Resuming dunning campaign", "OrgId", orgId, "CampaignId", campaignId)

	campaign, err := a.dunningService.ResumeCampaign(ctx, interfaces.ResumeDunningCampaignInput{
		OrgId:  orgId,
		Id:     campaignId,
		Reason: "manual_resume",
	})
	if err != nil {
		logger.Error("Failed to resume dunning campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	return campaign, nil
}

// CancelDunningCampaign cancels a dunning campaign
func (a *DunningActivities) CancelDunningCampaign(ctx context.Context, orgId string, campaignId string) (dunning.DunningCampaign, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Cancelling dunning campaign", "OrgId", orgId, "CampaignId", campaignId)

	campaign, err := a.dunningService.CancelCampaign(ctx, interfaces.CancelDunningCampaignInput{
		OrgId:  orgId,
		Id:     campaignId,
		Reason: "manual_cancel",
	})
	if err != nil {
		logger.Error("Failed to cancel dunning campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	return campaign, nil
}

// HandleSubscriptionStateChanged handles a subscription state change
func (a *DunningActivities) HandleSubscriptionStateChanged(ctx context.Context, orgId string, campaignId string, data interface{}) (dunning.DunningCampaign, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Handling subscription state change", "OrgId", orgId, "CampaignId", campaignId)

	// Get the campaign
	campaign, err := a.dunningService.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		logger.Error("Failed to find dunning campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Extract subscription state change data
	stateChange, ok := data.(interfaces.SubscriptionStateChangedInput)
	if !ok {
		logger.Error("Failed to convert data to SubscriptionStateChangedInput")
		return campaign, nil
	}

	// Handle different state changes
	switch stateChange.NewStatus {
	case entities.SubscriptionStatusCancelled:
		// If subscription is cancelled, cancel the campaign
		return a.CancelDunningCampaign(ctx, orgId, campaignId)
	case entities.SubscriptionStatusPaused:
		// If subscription is paused, pause the campaign
		return a.PauseDunningCampaign(ctx, orgId, campaignId)
	case entities.SubscriptionStatusActive:
		// If subscription is activated and campaign is paused, resume the campaign
		if campaign.Status == dunning.DunningStatusPaused {
			return a.ResumeDunningCampaign(ctx, orgId, campaignId)
		}
	}

	return campaign, nil
}

// ExecuteDunningAttempt executes a dunning payment attempt
func (a *DunningActivities) ExecuteDunningAttempt(ctx context.Context, orgId string, campaignId string, attemptType dunning.DunningAttemptType) (dunning.DunningAttempt, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing dunning attempt", "OrgId", orgId, "CampaignId", campaignId, "AttemptType", attemptType)

	// Trigger a manual attempt
	attempt, err := a.dunningService.TriggerManualAttempt(ctx, interfaces.TriggerManualAttemptInput{
		OrgId:       orgId,
		CampaignId:  campaignId,
		TriggeredBy: string(attemptType),
	})
	if err != nil {
		logger.Error("Failed to trigger manual attempt", "Error", err.Error())
		return dunning.DunningAttempt{}, err
	}

	return attempt, nil
}

// UpdateCampaignWithAttemptResult updates a campaign with an attempt result and handles all business logic
func (a *DunningActivities) UpdateCampaignWithAttemptResult(ctx context.Context, attempt dunning.DunningAttempt, config dunning.DunningConfig, attemptContext AttemptContext) (dunning.DunningCampaign, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating campaign with attempt result",
		"OrgId", attempt.OrgId,
		"CampaignId", attempt.DunningCampaignId,
		"AttemptId", attempt.Id,
		"Status", attempt.Status,
		"AttemptType", attempt.AttemptType,
		"AttemptNumber", attemptContext.AttemptNumber)

	// Get the current campaign
	campaign, err := a.dunningService.FindCampaignById(ctx, attempt.OrgId, attempt.DunningCampaignId)
	if err != nil {
		logger.Error("Failed to find dunning campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Update campaign statistics
	campaign.LastAttemptAt = attempt.AttemptedAt
	campaign.TotalAttempts++
	if attempt.AttemptType == dunning.DunningAttemptTypeImmediate {
		campaign.ImmediateAttempts++
	} else if attempt.AttemptType == dunning.DunningAttemptTypeProgressive {
		campaign.ProgressiveAttempts++
	}
	campaign.UpdatedAt = time.Now().UTC()

	// Handle successful payment
	if attempt.Status == payments.PaymentStatusSucceeded {
		logger.Info("Payment attempt succeeded, handling recovery")

		// Mark campaign as recovered
		campaign.Status = dunning.DunningStatusRecovered
		campaign.RecoveryMethod = string(attempt.AttemptType)
		campaign.RecoveredAmount = campaign.FailedAmount
		campaign.RecoveredAt = time.Now().UTC()
		campaign.CompletedAt = time.Now().UTC()

		// Update the campaign in the database
		updatedCampaign, err := a.dunningService.UpdateCampaign(ctx, attempt.OrgId, campaign)
		if err != nil {
			logger.Error("Failed to update campaign", "Error", err.Error())
			return dunning.DunningCampaign{}, err
		}

		// Reactivate subscription if it was suspended
		if attemptContext.WasSubscriptionSuspended {
			err = a.ReactivateSubscription(ctx, attempt.OrgId, campaign.SubscriptionId)
			if err != nil {
				logger.Error("Failed to reactivate subscription", "Error", err.Error())
				// Continue anyway, as the payment succeeded
			}
		} else {
			// Update subscription to active status and clear dunning metadata
			err = a.updateSubscriptionForRecovery(ctx, attempt.OrgId, campaign.SubscriptionId)
			if err != nil {
				logger.Error("Failed to update subscription for recovery", "Error", err.Error())
				// Continue anyway, as the payment succeeded
			}
		}

		// Publish recovery event
		event := topic.NewDunningCampaignEvent(updatedCampaign)
		err = a.pubsub.Publish(updatedCampaign.OrgId, topic.DunningCampaignRecovered, event)
		if err != nil {
			logger.Error("Failed to publish dunning campaign recovered event", "Error", err.Error())
		}

		return updatedCampaign, nil
	}

	// Handle failed payment - check escalation rules
	logger.Info("Payment attempt failed, checking escalation rules",
		"AttemptNumber", attemptContext.AttemptNumber,
		"FailureReason", attempt.FailureReason,
		"FailureCode", attempt.FailureCode)

	// Check if we need to suspend subscription
	shouldSuspend := attemptContext.AttemptNumber >= config.EscalationRules.SuspendAfterAttempt &&
		!attemptContext.WasSubscriptionSuspended &&
		attempt.AttemptType == dunning.DunningAttemptTypeProgressive

	if shouldSuspend {
		logger.Info("Suspending subscription due to escalation rules", "AttemptNumber", attemptContext.AttemptNumber)
		//err = a.SuspendSubscription(ctx, attempt.OrgId, campaign.SubscriptionId)
		//if err != nil {
		//	logger.Error("Failed to suspend subscription", "Error", err.Error())
		//}
	}

	// Check if we need to cancel subscription (final failure)
	shouldCancel := attemptContext.AttemptNumber >= config.EscalationRules.CancelAfterAttempt &&
		attempt.AttemptType == dunning.DunningAttemptTypeProgressive

	if shouldCancel {
		logger.Info("Cancelling subscription due to escalation rules", "AttemptNumber", attemptContext.AttemptNumber)

		// Cancel subscription
		err = a.CancelSubscription(ctx, attempt.OrgId, campaign.SubscriptionId)
		if err != nil {
			logger.Error("Failed to cancel subscription", "Error", err.Error())
		}

		// Mark campaign as failed
		campaign.Status = dunning.DunningStatusFailed
		campaign.FinalFailureReason = "max_attempts_reached"
		campaign.CompletedAt = time.Now().UTC()

		// Update the campaign in the database
		updatedCampaign, err := a.dunningService.UpdateCampaign(ctx, attempt.OrgId, campaign)
		if err != nil {
			logger.Error("Failed to update campaign", "Error", err.Error())
			return dunning.DunningCampaign{}, err
		}

		// Publish failed event
		event := topic.NewDunningCampaignEvent(updatedCampaign)
		err = a.pubsub.Publish(updatedCampaign.OrgId, topic.DunningCampaignFailed, event)
		if err != nil {
			logger.Error("Failed to publish dunning campaign failed event", "Error", err.Error())
		}

		return updatedCampaign, nil
	}

	// For regular failures, just update the campaign
	updatedCampaign, err := a.dunningService.UpdateCampaign(ctx, attempt.OrgId, campaign)
	if err != nil {
		logger.Error("Failed to update campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Publish attempt failed event
	event := topic.NewDunningAttemptEvent(attempt, updatedCampaign, shouldSuspend, shouldCancel)
	err = a.pubsub.Publish(attempt.OrgId, topic.DunningAttemptFailed, event)
	if err != nil {
		logger.Error("Failed to publish dunning attempt failed event", "Error", err.Error())
	}

	return updatedCampaign, nil
}

// updateSubscriptionForRecovery updates subscription status when payment recovery occurs
func (a *DunningActivities) updateSubscriptionForRecovery(ctx context.Context, orgId string, subscriptionId string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating subscription for recovery", "OrgId", orgId, "SubscriptionId", subscriptionId)

	// Update subscription
	_, err := a.subscriptionService.Activate(ctx, orgId, subscriptionId)
	if err != nil {
		logger.Error("Failed to update subscription", "err", err.Error())
		return err
	}

	return nil
}

// MarkCampaignRecovered marks a campaign as recovered
func (a *DunningActivities) MarkCampaignRecovered(ctx context.Context, orgId string, campaignId string, recoveryMethod string) (dunning.DunningCampaign, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Marking campaign as recovered", "OrgId", orgId, "CampaignId", campaignId, "RecoveryMethod", recoveryMethod)

	// Get the campaign
	campaign, err := a.dunningService.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		logger.Error("Failed to find dunning campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Update campaign status
	campaign.Status = dunning.DunningStatusRecovered
	campaign.RecoveryMethod = recoveryMethod
	campaign.RecoveredAmount = campaign.FailedAmount
	campaign.RecoveredAt = time.Now().UTC()
	campaign.CompletedAt = time.Now().UTC()

	// Update subscription
	subscription, err := a.subscriptionService.FindById(ctx, orgId, campaign.SubscriptionId)
	if err != nil {
		logger.Error("Failed to find subscription", "Error", err.Error())
	} else {
		// Update subscription
		_, err = a.subscriptionService.Activate(ctx, orgId, subscription.Id)
		if err != nil {
			logger.Error("Failed to update subscription", "err", err.Error())
		}
	}

	// Publish event
	event := topic.NewDunningCampaignEvent(campaign)
	err = a.pubsub.Publish(orgId, topic.DunningCampaignRecovered, event)
	if err != nil {
		logger.Error("Failed to publish dunning campaign recovered event", "Error", err.Error())
	}

	return campaign, nil
}

// MarkCampaignFailed marks a campaign as failed
func (a *DunningActivities) MarkCampaignFailed(ctx context.Context, orgId string, campaignId string, failureReason string) (dunning.DunningCampaign, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Marking campaign as failed", "OrgId", orgId, "CampaignId", campaignId, "FailureReason", failureReason)

	// Get the campaign
	campaign, err := a.dunningService.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		logger.Error("Failed to find dunning campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Update campaign status
	campaign.Status = dunning.DunningStatusFailed
	campaign.FinalFailureReason = failureReason
	campaign.CompletedAt = time.Now().UTC()

	// Update subscription
	_, err = a.subscriptionService.FindById(ctx, orgId, campaign.SubscriptionId)
	if err != nil {
		logger.Error("Failed to find subscription", "Error", err.Error())
	} else {
		// TODO
		// maybe update the subscription here.  Dunning failed and the subscription is in past_due state.
	}

	// Publish event
	event := topic.NewDunningCampaignEvent(campaign)
	err = a.pubsub.Publish(orgId, topic.DunningCampaignFailed, event)
	if err != nil {
		logger.Error("Failed to publish dunning campaign failed event", "Error", err.Error())
	}

	return campaign, nil
}

// SendDunningCommunication sends a dunning communication
func (a *DunningActivities) SendDunningCommunication(ctx context.Context, orgId string, campaignId string, attemptNumber int) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending dunning communication", "OrgId", orgId, "CampaignId", campaignId, "AttemptNumber", attemptNumber)

	// Get the campaign
	campaign, err := a.dunningService.FindCampaignById(ctx, orgId, campaignId)
	if err != nil {
		logger.Error("Failed to find dunning campaign", "Error", err.Error())
		return err
	}

	// Publish event for notification service to handle
	event := map[string]interface{}{
		"org_id":          orgId,
		"campaign_id":     campaignId,
		"customer_id":     campaign.CustomerId,
		"subscription_id": campaign.SubscriptionId,
		"attempt_number":  attemptNumber,
		"timestamp":       time.Now().UTC(),
	}
	err = a.pubsub.Publish(orgId, topic.DunningCommunicationSent, event)
	if err != nil {
		logger.Error("Failed to publish dunning communication event", "Error", err.Error())
		return err
	}

	return nil
}

// ReactivateSubscription reactivates a suspended subscription
func (a *DunningActivities) ReactivateSubscription(ctx context.Context, orgId string, subscriptionId string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Reactivating subscription", "OrgId", orgId, "SubscriptionId", subscriptionId)

	// Update subscription
	_, err := a.subscriptionService.Activate(ctx, orgId, subscriptionId)
	if err != nil {
		logger.Error("Failed to update subscription", "err", err.Error())
		return err
	}

	// Publish event
	event := map[string]interface{}{
		"org_id":          orgId,
		"subscription_id": subscriptionId,
		"timestamp":       time.Now().UTC(),
	}
	err = a.pubsub.Publish(orgId, topic.DunningSubscriptionReactivated, event)
	if err != nil {
		logger.Error("Failed to publish subscription reactivated event", "err", err.Error())
	}

	return nil
}

// CancelSubscription cancels a subscription
func (a *DunningActivities) CancelSubscription(ctx context.Context, orgId string, subscriptionId string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Cancelling subscription", "OrgId", orgId, "SubscriptionId", subscriptionId)

	// Update subscription
	_, err := a.subscriptionService.CancelSubscription(ctx, subscriptions.CancelSubscriptionInput{
		OrgId: orgId,
		Id:    subscriptionId,
	})
	if err != nil {
		logger.Error("Failed to cancel subscription", "Error", err.Error())
		return err
	}

	return nil
}
