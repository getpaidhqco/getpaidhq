package workflows

import (
	"fmt"
	"payloop/internal/domain/entities"
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

// DunningWorkflow is a Temporal workflow that manages the dunning process for a failed payment.
// It doesn't handle technical failures, that is owned by the SubscriptionWorkflow. If a dunning
// workflow is started, it means that the payment was successfully attempted but failed
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
	var subscription entities.Subscription

	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute * 5,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second * 1,
				BackoffCoefficient: 2.0,
				MaximumInterval:    time.Minute * 5,
				MaximumAttempts:    5,
			},
		}),
		a.CreateDunningCampaign, input).
		Get(ctx, &campaign)
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
				}), a.HandleDunningChargeResult, campaign.OrgId, campaign.Id).Get(ctx, nil)
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

	// Phase 1: Progressive Retries with customer communication

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

		// Activity:
		// Send customer communication before the attempt
		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 5,
			}),
			a.SendDunningCommunication, campaign.OrgId, campaign.Id, i+1).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to send dunning communication", "Error", err.Error())
		}

		/*
		 * ACTIVITY: Charge the customer for the billing period
		 */
		logger.Info("Subscription is active, processing cycle")
		// Charge the customer
		var chargeResult payments.ChargeResult
		chargeCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Second * 60,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Minute * 2,
				BackoffCoefficient: 1.2,
			},
		})
		err = workflow.ExecuteActivity(chargeCtx,
			a.ProcessRetryCharge,
			subscription).
			Get(chargeCtx, &chargeResult)
		if err != nil {
			// a system error occurred when attempting to charge the customer
			// This shouldn't be reached because of the retry policy which will retry forever
			// The workflow can be stopped using updates
			// TODO report this error
			logger.Error("ProcessRetryCharge failed completely, ending workflow", "Error", err.Error())
			return campaign, err
		}

		// Charge is completed
		// If payment status is Pending, then we must wait for a webhook to complete the payment
		if chargeResult.Status == payments.PaymentStatusPending {
			// Wait for the webhook
			selector := workflow.NewSelector(ctx)
			webhookChan := workflow.GetSignalChannel(ctx, "webhook-signal")

			selector.AddReceive(webhookChan, func(c workflow.ReceiveChannel, more bool) {
				logger.Info("Received webhook signal")
				c.Receive(ctx, &chargeResult)
			})

			// Wait for either the webhook or a timeout
			timeout := workflow.NewTimer(ctx, 24*time.Hour)
			selector.AddFuture(timeout, func(f workflow.Future) {
				// Handle timeout
				logger.Error("Timeout waiting for payment webhook", "Error", "")
			})

			selector.Select(ctx)
		}

		// Activity: HandleDunningChargeResult
		// This updates the Campaign and creates a DunningAttempt based on the charge result.
		var handleChargeResult activities.HandleChargeAttemptResult
		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 5,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Minute * 2,
					BackoffCoefficient: 1.2,
				},
			}),
			a.HandleDunningChargeResult, campaign, chargeResult, config).
			Get(ctx, &handleChargeResult)
		if err != nil {
			logger.Error("Error calling HandleDunningChargeResult", "Error", err.Error())
			continue
		}

		campaign = handleChargeResult.Campaign
		subscription = handleChargeResult.Subscription

		// Activity: HandleDunningChargeResult
		// This updates the Campaign and creates a DunningAttempt based on the charge result.
		var result activities.HandleChargeAttemptResult
		err = workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute * 5,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    time.Minute * 2,
					BackoffCoefficient: 1.2,
				},
			}),
			a.HandleDunningChargeResult, campaign, chargeResult, config).
			Get(ctx, &result)
		if err != nil {
			logger.Error("Error calling HandleDunningChargeResult", "Error", err.Error())
			continue
		}

		campaign = result.Campaign
		subscription = result.Subscription

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

	logger.Info("DunningWorkflow completed", "Status", campaign.Status)
	return campaign, nil
}
