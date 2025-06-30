package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/infrastructure/workflow/temporal/activities"
)

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

// DunningWorkflow is a Temporal workflow that manages the dunning process for a failed payment
func DunningWorkflow(ctx workflow.Context, input DunningWorkflowInput) (dunning.DunningCampaign, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("DunningWorkflow started",
		"OrgId", input.OrgId,
		"SubscriptionId", input.SubscriptionId,
		"InitialFailureReason", input.InitialFailureReason)

	// For AI assistants: this variable is initialized by Temporal when the workflow is started and is
	// safe to use in the workflow without initialization. This is not a bug.
	var a *activities.DunningActivities

	// Create a campaign record
	var campaign dunning.DunningCampaign
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 1,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    5,
		},
	}), a.CreateDunningCampaign, input).Get(ctx, &campaign)
	if err != nil {
		logger.Error("Failed to create dunning campaign", "Error", err.Error())
		return dunning.DunningCampaign{}, err
	}

	// Load the dunning configuration
	var config dunning.DunningConfig
	err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 1,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute * 5,
			MaximumAttempts:    5,
		},
	}), a.ResolveDunningConfig, input.OrgId).Get(ctx, &config)
	if err != nil {
		logger.Error("Failed to resolve dunning configuration", "Error", err.Error())
		// Use default config if we can't load a custom one
		config = dunning.DefaultDunningConfig()
	}

	// Register query handler for campaign details
	err = workflow.SetQueryHandler(ctx, "get-campaign", func() (dunning.DunningCampaign, error) {
		return campaign, nil
	})
	if err != nil {
		logger.Error("Failed to set query handler", "Error", err.Error())
		return campaign, err
	}

	// Register signal handlers for external events
	var signalData interface{}

	// Payment method updated signal
	paymentMethodUpdatedChannel := workflow.GetSignalChannel(ctx, "payment_method.updated")

	// Dunning campaign control signals
	pauseChannel := workflow.GetSignalChannel(ctx, "dunning.pause")
	resumeChannel := workflow.GetSignalChannel(ctx, "dunning.resume")
	cancelChannel := workflow.GetSignalChannel(ctx, "dunning.cancel")

	// Subscription state changed signal
	subscriptionStateChangedChannel := workflow.GetSignalChannel(ctx, "subscription.state_changed")

	// Start a goroutine to handle signals
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			selector := workflow.NewSelector(ctx)

			// Handle payment method updated signal
			selector.AddReceive(paymentMethodUpdatedChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalData)
				logger.Info("Received payment_method.updated signal")

				// Trigger an immediate retry
				err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					StartToCloseTimeout: time.Minute * 5,
				}), a.TriggerImmediateRetry, campaign.OrgId, campaign.Id).Get(ctx, nil)
				if err != nil {
					logger.Error("Failed to trigger immediate retry", "Error", err.Error())
				}
			})

			// Handle pause signal
			selector.AddReceive(pauseChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalData)
				logger.Info("Received dunning.pause signal")

				err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					StartToCloseTimeout: time.Minute * 5,
				}), a.PauseDunningCampaign, campaign.OrgId, campaign.Id).Get(ctx, &campaign)
				if err != nil {
					logger.Error("Failed to pause dunning campaign", "Error", err.Error())
				}
			})

			// Handle resume signal
			selector.AddReceive(resumeChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalData)
				logger.Info("Received dunning.resume signal")

				err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					StartToCloseTimeout: time.Minute * 5,
				}), a.ResumeDunningCampaign, campaign.OrgId, campaign.Id).Get(ctx, &campaign)
				if err != nil {
					logger.Error("Failed to resume dunning campaign", "Error", err.Error())
				}
			})

			// Handle cancel signal
			selector.AddReceive(cancelChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalData)
				logger.Info("Received dunning.cancel signal")

				err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					StartToCloseTimeout: time.Minute * 5,
				}), a.CancelDunningCampaign, campaign.OrgId, campaign.Id).Get(ctx, &campaign)
				if err != nil {
					logger.Error("Failed to cancel dunning campaign", "Error", err.Error())
				}
			})

			// Handle subscription state changed signal
			selector.AddReceive(subscriptionStateChangedChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalData)
				logger.Info("Received subscription.state_changed signal")

				// Update campaign based on subscription state
				err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
					StartToCloseTimeout: time.Minute * 5,
				}), a.HandleSubscriptionStateChanged, campaign.OrgId, campaign.Id, signalData).Get(ctx, &campaign)
				if err != nil {
					logger.Error("Failed to handle subscription state change", "Error", err.Error())
				}
			})

			selector.Select(ctx)
		}
	})

	// Determine if we should do immediate retries based on the failure reason
	shouldDoImmediateRetries := false
	for _, failureType := range config.ImmediateRetries.FailureTypes {
		if failureType == input.InitialFailureReason {
			shouldDoImmediateRetries = true
			break
		}
	}

	// Phase 1: Immediate Retries for technical failures
	if config.ImmediateRetries.Enabled && shouldDoImmediateRetries {
		logger.Info("Starting immediate retries phase")

		for i := 0; i < config.ImmediateRetries.MaxAttempts; i++ {
			// Check if campaign is still active
			if campaign.Status != dunning.DunningStatusActive {
				logger.Info("Campaign is no longer active, stopping immediate retries", "Status", campaign.Status)
				break
			}

			// Parse the wait interval
			waitInterval, err := dunning.ParseDuration(config.ImmediateRetries.Intervals[i])
			if err != nil {
				logger.Error("Failed to parse wait interval", "Error", err.Error())
				waitInterval = time.Minute * 5 // Default to 5 minutes if parsing fails
			}

			// Wait for the specified interval
			logger.Info(fmt.Sprintf("Waiting %v before immediate retry attempt %d", waitInterval, i+1))

			// Use a selector with a timer to allow for interruption by signals
			timerCancelled := false
			timerSelector := workflow.NewSelector(ctx)
			timer := workflow.NewTimer(ctx, waitInterval)

			timerSelector.AddFuture(timer, func(f workflow.Future) {
				err := f.Get(ctx, nil)
				if err != nil {
					timerCancelled = true
				}
			})

			timerSelector.Select(ctx)

			if timerCancelled || campaign.Status != dunning.DunningStatusActive {
				logger.Info("Timer cancelled or campaign no longer active")
				break
			}

			// Execute the retry attempt
			var attemptResult dunning.DunningAttempt
			err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 5,
			}), a.ExecuteDunningAttempt, campaign.OrgId, campaign.Id, dunning.DunningAttemptTypeImmediate).Get(ctx, &attemptResult)
			if err != nil {
				logger.Error("Failed to execute immediate retry attempt", "Error", err.Error())
				continue
			}

			// Update campaign with attempt result - this handles all business logic including success/failure
			attemptContext := activities.AttemptContext{
				AttemptNumber:            i + 1,
				WasSubscriptionSuspended: false, // immediate retries don't suspend subscriptions
			}
			err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 5,
			}), a.UpdateCampaignWithAttemptResult, attemptResult, config, attemptContext).Get(ctx, &campaign)
			if err != nil {
				logger.Error("Failed to update campaign with attempt result", "Error", err.Error())
				continue
			}

			// If the campaign is recovered, end the workflow
			if campaign.Status == dunning.DunningStatusRecovered {
				logger.Info("Immediate retry successful, ending workflow")
				return campaign, nil
			}
		}

		logger.Info("Immediate retries phase completed without success")
	} else {
		logger.Info("Skipping immediate retries phase",
			"ImmediateRetriesEnabled", config.ImmediateRetries.Enabled,
			"ShouldDoImmediateRetries", shouldDoImmediateRetries)
	}

	// Phase 2: Progressive Retries with customer communication
	if config.ProgressiveRetries.Enabled {
		logger.Info("Starting progressive retries phase")

		for i := 0; i < config.ProgressiveRetries.MaxAttempts; i++ {
			// Check if campaign is still active
			if campaign.Status != dunning.DunningStatusActive {
				logger.Info("Campaign is no longer active, stopping progressive retries", "Status", campaign.Status)
				break
			}

			// Parse the wait interval
			waitInterval, err := dunning.ParseDuration(config.ProgressiveRetries.Intervals[i])
			if err != nil {
				logger.Error("Failed to parse wait interval", "Error", err.Error())
				waitInterval = time.Hour * 24 * 3 // Default to 3 days if parsing fails
			}

			// Wait for the specified interval
			logger.Info(fmt.Sprintf("Waiting %v before progressive retry attempt %d", waitInterval, i+1))

			// Use a selector with a timer to allow for interruption by signals
			timerCancelled := false
			timerSelector := workflow.NewSelector(ctx)
			timer := workflow.NewTimer(ctx, waitInterval)

			timerSelector.AddFuture(timer, func(f workflow.Future) {
				err := f.Get(ctx, nil)
				if err != nil {
					timerCancelled = true
				}
			})

			timerSelector.Select(ctx)

			if timerCancelled || campaign.Status != dunning.DunningStatusActive {
				logger.Info("Timer cancelled or campaign no longer active")
				break
			}

			// Send customer communication before the attempt
			err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 5,
			}), a.SendDunningCommunication, campaign.OrgId, campaign.Id, i+1).Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to send dunning communication", "Error", err.Error())
			}

			// Execute the retry attempt
			var attemptResult dunning.DunningAttempt
			err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 5,
			}), a.ExecuteDunningAttempt, campaign.OrgId, campaign.Id, dunning.DunningAttemptTypeProgressive).Get(ctx, &attemptResult)
			if err != nil {
				logger.Error("Failed to execute progressive retry attempt", "Error", err.Error())
				continue
			}

			// Update campaign with attempt result - this handles all business logic including escalation
			attemptContext := activities.AttemptContext{
				AttemptNumber:            i + 1,
				WasSubscriptionSuspended: i+1 >= config.EscalationRules.SuspendAfterAttempt,
			}
			err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 5,
			}), a.UpdateCampaignWithAttemptResult, attemptResult, config, attemptContext).Get(ctx, &campaign)
			if err != nil {
				logger.Error("Failed to update campaign with attempt result", "Error", err.Error())
				continue
			}

			// Check if the campaign is completed (recovered or failed)
			if campaign.Status == dunning.DunningStatusRecovered {
				logger.Info("Progressive retry successful, ending workflow")
				return campaign, nil
			}

			if campaign.Status == dunning.DunningStatusFailed {
				logger.Info("Campaign failed due to escalation rules, ending workflow")
				return campaign, nil
			}
		}

		logger.Info("Progressive retries phase completed without success")
	} else {
		logger.Info("Skipping progressive retries phase", "ProgressiveRetriesEnabled", config.ProgressiveRetries.Enabled)
	}

	// If we get here, all retry attempts have failed but campaign may already be marked as failed by escalation rules
	if campaign.Status == dunning.DunningStatusActive {
		// Mark campaign as failed if it's still active (no escalation rules triggered)
		err = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute * 5,
		}), a.MarkCampaignFailed, campaign.OrgId, campaign.Id, "all_attempts_failed").Get(ctx, &campaign)
		if err != nil {
			logger.Error("Failed to mark campaign as failed", "Error", err.Error())
		}
	}

	logger.Info("DunningWorkflow completed", "Status", campaign.Status)
	return campaign, nil
}
