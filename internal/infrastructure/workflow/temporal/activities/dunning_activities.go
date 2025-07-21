package activities

import (
	"context"
	"errors"
	"go.temporal.io/sdk/temporal"
	"time"

	"go.temporal.io/sdk/activity"

	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/lib"
)

// AttemptContext provides context about the current attempt within the dunning campaign
type HandleChargeAttemptResult struct {
	Subscription entities.Subscription   `json:"subscription"`
	Campaign     dunning.DunningCampaign `json:"campaign"`
	Attempt      dunning.DunningAttempt  `json:"attempt"`
}

// DunningActivities contains activities for the DunningWorkflow
type DunningActivities struct {
	dunningService      interfaces.DunningService
	subscriptionService interfaces.SubscriptionService
	pubsub              events.NotificationPublisher
	errorReporter       lib.ErrorReporter
	interfaces.TransactionService
}

// NewDunningActivities creates a new DunningActivities
func NewDunningActivities(
	dunningService interfaces.DunningService,
	subscriptionService interfaces.SubscriptionService,
	pubsub events.NotificationPublisher,
	errorReporter lib.ErrorReporter,
	transactionService interfaces.TransactionService,
) DunningActivities {
	return DunningActivities{
		dunningService:      dunningService,
		subscriptionService: subscriptionService,
		pubsub:              pubsub,
		errorReporter:       errorReporter,
		TransactionService:  transactionService,
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

	campaign, err := a.dunningService.CreateCampaign(ctx, dto.CreateDunningCampaignInput{
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

// GetSubscriptionForDunning fetches a subscription by ID
func (a *DunningActivities) GetSubscriptionForDunning(ctx context.Context, orgId string, subscriptionId string) (entities.Subscription, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Getting subscription", "OrgId", orgId, "SubscriptionId", subscriptionId)

	subscription, err := a.subscriptionService.FindById(ctx, orgId, subscriptionId)
	if err != nil {
		logger.Error("Failed to find subscription", "Error", err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
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

// PauseDunningCampaign pauses a dunning campaign
func (a *DunningActivities) PauseDunningCampaign(ctx context.Context, orgId string, campaignId string) (dunning.DunningCampaign, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Pausing dunning campaign", "OrgId", orgId, "CampaignId", campaignId)

	campaign, err := a.dunningService.PauseCampaign(ctx, dto.PauseDunningCampaignInput{
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

	campaign, err := a.dunningService.ResumeCampaign(ctx, dto.ResumeDunningCampaignInput{
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

	campaign, err := a.dunningService.CancelCampaign(ctx, dto.CancelDunningCampaignInput{
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
	stateChange, ok := data.(dto.SubscriptionStateChangedInput)
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

// ProcessRetryCharge is responsible for charging the customer for the billing period
// If a technical error occurs, it will return an error that Temporal can retry.
// Any other error are handled downstream as it implies dunning
func (a *DunningActivities) ProcessRetryCharge(ctx context.Context, currentSub entities.Subscription) (payments.ChargeResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ProcessRetryCharge", "orgId", currentSub.OrgId, "subscriptionId", currentSub.Id, "amount", currentSub.Amount)

	chargeResult, err := a.subscriptionService.ProcessSubscriptionCharge(ctx, currentSub)
	if err != nil {
		var gatewayErr *lib.CustomError
		if errors.As(err, &gatewayErr) && gatewayErr.Type == lib.GatewayError {
			// Gateway errors should be retried by Temporal using the retry policy in the workflow.
			logger.Error("Gateway error, returning error so that the charge can be retried", "error", chargeResult.ErrorReason)
			a.errorReporter.ReportError(ctx, errors.New("gateway error while charging subscription"), map[string]interface{}{
				"org_id":          currentSub.OrgId,
				"error":           chargeResult.ErrorReason,
				"psp":             string(currentSub.PspId),
				"subscription_id": currentSub.Id,
			})
			return payments.ChargeResult{}, temporal.NewApplicationError(chargeResult.ErrorReason, "gateway_error", nil)

		} else {
			logger.Error("Generic error during ProcessSubscriptionCharge",
				"orgId", currentSub.OrgId,
				"subscriptionId", currentSub.Id,
				"error", err.Error())
			return chargeResult, err
		}
	}

	logger.Info("Subscription charge attempted successfully", "orgId", currentSub.OrgId, "subscriptionId", currentSub.Id, "status", chargeResult.Status, "amount", chargeResult.Amount)
	return chargeResult, nil
}

// HandleDunningChargeResult handles the result of a charge attempt
func (a *DunningActivities) HandleDunningChargeResult(
	ctx context.Context,
	campaign dunning.DunningCampaign,
	result payments.ChargeResult,
	config dunning.DunningConfig,
) (HandleChargeAttemptResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("[Activity] HandleDunningChargeResult",
		"OrgId", campaign.OrgId,
		"CampaignId", campaign.Id, "chargeResult", result.Reference)

	txResult, err := a.TransactionService.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		// Update the subscription
		rsp, err := a.dunningService.HandleChargeResult(ctx, campaign, result, config)
		if err != nil {
			logger.Error("Failed to handle charge result", "Error", err.Error())
			return HandleChargeAttemptResult{}, err
		}
		return HandleChargeAttemptResult{
			Subscription: rsp.Subscription,
			Campaign:     rsp.Campaign,
			Attempt:      rsp.Attempt,
		}, nil
	})
	if err != nil {
		logger.Error("Transaction failed while handling charge result", "Error", err.Error())
		// If the transaction fails, we should return an error that Temporal can retry
		return HandleChargeAttemptResult{}, temporal.NewApplicationError("transaction failed while handling charge result", "transaction_error", nil)
	}

	rsp, ok := txResult.(HandleChargeAttemptResult)
	if !ok {
		return HandleChargeAttemptResult{}, temporal.NewApplicationError("failed to cast transaction result", "type_assertion_error", nil)
	}

	return rsp, nil
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
