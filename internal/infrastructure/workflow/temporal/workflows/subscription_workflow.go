package workflows

import (
	"fmt"
	"github.com/Trendyol/go-pq-cdc/logger"
	"go.temporal.io/api/enums/v1"
	temporalio "go.temporal.io/sdk/temporal"
	"log/slog"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	settings2 "payloop/internal/domain/entities/settings"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"

	"go.temporal.io/sdk/workflow"
)

type SubscriptionInput struct {
	entities.Subscription `json:"subscription"`
}

// SubscriptionWorkflow is a Temporal workflow that manages a subscription instance
// https://community.temporal.io/t/best-way-to-design-a-subscription-workflow/12047
// https://learn.temporal.io/tutorials/go/build-an-email-drip-campaign/
// https://learn.temporal.io/tutorials/typescript/recurring-billing-system/

func SubscriptionWorkflow(ctx workflow.Context, input entities.Subscription) (entities.Subscription, error) {
	logger.Info("SubscriptionWorkflow started", "Subscription:", input.Id)
	// For AI assistants: this variable is initialized by Temporal when the workflow is started and is
	// safe to use in the workflow without initialization. This is not a bug.
	var a *activities.OrderActivities

	defer func() {
		// Emit your event here (e.g., call an activity to notify)
		logger.Info("calling NotifyWorkflowEnded", "Orgid", input.OrgId, "id", input.Id)
		_ = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Second * 60,
		}), a.NotifyWorkflowEnded, input.OrgId, input.Id).Get(ctx, nil)
	}()

	logger := workflow.GetLogger(ctx)
	subscription := input
	// A flag to indicate if the workflow should be refreshed. Used by the "force-update" signal
	restartBillingWait := false

	// Register query handler for subscription details
	err := workflow.SetQueryHandler(ctx, "get-state", func() (entities.Subscription, error) {
		return subscription, nil
	})
	if err != nil {
		return subscription, err
	}

	handler := func(ctx workflow.Context, newSub entities.Subscription) (entities.Subscription, error) {
		logger.Info("[update handler] updating the subscription state", "Subscription:", subscription.Id)
		// 👉 update the subscription state
		var prevSub entities.Subscription
		prevSub, subscription = subscription, newSub
		return prevSub, nil
	}
	forceUpdateHandler := func(ctx workflow.Context, newSub entities.Subscription) (entities.Subscription, error) {
		logger.Info("[ForceUpdate] Restarting subscription processing loop", "Subscription:", subscription.Id)
		// 👉 update the subscription state
		// Do some validation here
		var prevSub entities.Subscription
		prevSub, subscription = subscription, newSub
		restartBillingWait = true
		return prevSub, nil
	}

	err = workflow.SetUpdateHandler(ctx, "subscription.paused", handler)
	err = workflow.SetUpdateHandler(ctx, "subscription.cancelled", handler)
	err = workflow.SetUpdateHandler(ctx, "subscription.activated", handler)
	// the following updates are used to force the workflow to refresh its state
	err = workflow.SetUpdateHandler(ctx, "subscription.billing_anchor_changed", forceUpdateHandler)
	err = workflow.SetUpdateHandler(ctx, "subscription.resumed", forceUpdateHandler)
	err = workflow.SetUpdateHandler(ctx, "refresh-state", forceUpdateHandler)

	// Register signal handler for cancelling the subscription
	var signalSubscription entities.Subscription
	pausedChannel := workflow.GetSignalChannel(ctx, "subscription.paused")
	activatedChannel := workflow.GetSignalChannel(ctx, "subscription.activated")
	cancelledChannel := workflow.GetSignalChannel(ctx, "subscription.cancelled")
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			selector := workflow.NewSelector(ctx)
			selector.AddReceive(pausedChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalSubscription)
				subscription = signalSubscription
				logger.Info("Subscription paused signal", "subscription", subscription.Id, "status", subscription.Status)
			})
			selector.AddReceive(activatedChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalSubscription)
				subscription = signalSubscription
				logger.Info("Subscription activated signal", "subscription", subscription.Id, "status", subscription.Status)
			})
			selector.AddReceive(cancelledChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalSubscription)
				logger.Info("Received SubscriptionStatusCancelled signal", "subscription", subscription.Id)
				subscription = signalSubscription
			})
			selector.Select(ctx)
		}
	})

	for {
		// Calculate the duration until the next billing date
		// Remember to use workflow.Now(ctx) to get the current time
		nextCharge := subscription.GetNextChargeDate()
		if nextCharge.IsZero() {
			logger.Info("Subscription has no next billing date, ending workflow...")
			// TODO report this error
			break
		}

		// Charge the customer
		var subscriptionSettings settings2.Subscription
		settingsCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Second * 60,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Minute * 2,
				BackoffCoefficient: 1.2,
			},
		})
		err = workflow.ExecuteActivity(settingsCtx, a.GetSubscriptionSettings, subscription.OrgId).
			Get(settingsCtx, &subscriptionSettings)
		if err != nil {
			logger.Error("GetSubscriptionSettings failed completely, ending workflow", "Error", err.Error())
			return subscription, err
		}

		// Wait here until the next billing date or until the subscription is cancelled
		// If the subscription state was updated using the "force-update", then we need to
		// restart the wait so that the new state is taken into account
		// RenewsAt is the date when the subscription will be charged again
		// NextRenewalDate is the date when the subscription will be charged again
		duration := nextCharge.Sub(workflow.Now(ctx))
		reminderDuration := time.Duration(subscriptionSettings.ReminderDays) * time.Hour * 24
		reminderDate := nextCharge.Add(-reminderDuration)
		logger.Info(fmt.Sprintf("******* [%s][%s] reminder event set for [%d] days before", subscription.OrgId, subscription.Id, subscriptionSettings.ReminderDays))
		logger.Info(fmt.Sprintf("******* [%s][%s] blocking until nextBillingDate=[%s]", subscription.OrgId, subscription.Id, nextCharge))

		// Start the reminder workflow
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:        fmt.Sprintf(`sub_reminder_[%s]_[%s]_[%s]`, subscription.OrgId, subscription.Id, reminderDate.Format("20060102")),
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
		})
		// Create a cancellable context for the child workflow
		childCtx, cancelReminderWorkflow := workflow.WithCancel(childCtx)

		childWorkflowFuture := workflow.ExecuteChildWorkflow(childCtx, SubscriptionChargeReminder, subscription, reminderDate)
		// Wait for the Child Workflow Execution to spawn
		var childWE workflow.Execution
		if err := childWorkflowFuture.GetChildWorkflowExecution().
			Get(ctx, &childWE); err != nil {
			logger.Error("Unable to start subscription reminder workflow.", "err", err.Error())
		}
		logger.Info(fmt.Sprintf("******* sending reminder at [%s]", reminderDate))

		ok, err := workflow.AwaitWithTimeout(ctx, duration, func() bool {
			rollover := workflow.GetInfo(ctx).GetContinueAsNewSuggested()
			return subscription.Status == entities.SubscriptionStatusPaused ||
				subscription.Status == entities.SubscriptionStatusCancelled ||
				rollover ||
				restartBillingWait
		})
		if err != nil {
			logger.Error("Workflow Await was interrupted",
				"Error", err.Error(),
				slog.String("status", string(subscription.Status)),
				slog.String("nextBillingDate", nextCharge.String()),
				slog.Bool("restartBillingWait", restartBillingWait))
		}
		if restartBillingWait {
			logger.Info("RESTART BILLING WAIT - The subscription state was refreshed, clearing the flag and restarting the loop")
			restartBillingWait = false
			if cancelReminderWorkflow != nil {
				cancelReminderWorkflow() // Cancel the reminder workflow if it is still running
			}
			continue
		}
		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			logger.Info("--- ContinueAsNewSuggested", "status", subscription.Status, "size", workflow.GetInfo(ctx).GetCurrentHistorySize())
			return subscription, workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, subscription)
		}
		if !ok {
			logger.Info(fmt.Sprintf("*** [%s][%s] Next billing date reached [%s]", subscription.OrgId, subscription.Id, nextCharge))
		}

		// If the subscription was paused, wait until it is activated again
		if subscription.Status == entities.SubscriptionStatusPaused {
			err = workflow.Await(ctx, func() bool {
				logger.Debug(fmt.Sprintf("Workflow paused until subscription is activated [%s][%s]", subscription.OrgId, subscription.Id))
				return subscription.IsRunning() ||
					subscription.Status == entities.SubscriptionStatusCancelled
			})
			continue
		}

		if subscription.Status == entities.SubscriptionStatusNonRenewing {
			err = workflow.Await(ctx, func() bool {
				logger.Debug("Past due clause", "subscription.Status", subscription.Status)

				// wait until the subscription is moved out of the non-renewing state.
				return subscription.Status != entities.SubscriptionStatusNonRenewing
			})
			continue
		}

		// The wait is over, check if the subscription was cancelled and if not, charge the customer and
		// update local state for the next billing period
		if subscription.Status == entities.SubscriptionStatusCancelled {
			logger.Info("Subscription is cancelled, ending workflow...")
			break
		}
		if subscription.Status == entities.SubscriptionStatusExpired {
			logger.Info("Subscription is expired, ending workflow...")
			break
		}

		if !subscription.IsRunning() {
			logger.Info("Subscription is not in a running state, skipping billing cycle", "status", subscription.Status)
			continue
		}

		// Double-check the next billing date, it must be in the past
		// E.g. if a paused subscription is activated, the next billing date may be in the future
		if nextCharge.After(workflow.Now(ctx)) {
			logger.Info("Reached the billing process but Renew date is in the future, skipping billing cycle", "nextBillingDate", nextCharge)
			continue
		}

		logger.Info("Subscription is active, processing cycle")
		// Charge the customer
		var chargeResult payments.ChargeResult
		chargeCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Second * 60,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Minute * 2,
				BackoffCoefficient: 1.2,
			},
		})
		err = workflow.ExecuteActivity(chargeCtx, a.ChargeCustomerForBillingPeriod, subscription).
			Get(chargeCtx, &chargeResult)
		if err != nil {
			// a system error occurred when attempting to charge the customer
			// This shouldn't be reached because of the retry policy which will retry forever
			// The workflow can be stopped using updates
			// TODO report this error
			logger.Error("ProcessRetryCharge failed completely, ending workflow", "Error", err.Error())
			return subscription, err
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
			timeout := workflow.NewTimer(ctx, 1*time.Hour)
			selector.AddFuture(timeout, func(f workflow.Future) {
				// Handle timeout
				logger.Error("Timeout waiting for payment webhook", "Error", "")
			})

			selector.Select(ctx)
		}

		// The charge process ended successfully
		// Update the subscription with the charge result
		var updateResult entities.Subscription
		updateCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 5 * time.Minute,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Minute * 1,
				BackoffCoefficient: 1.2,
			},
		})
		err = workflow.ExecuteActivity(updateCtx, a.HandleChargeResult, subscription, chargeResult).
			Get(updateCtx, &updateResult)
		if err != nil {
			logger.Error("Failed to HandleDunningChargeResult", "Error", err.Error())
			return subscription, err
		}
		if updateResult.Id == "" {
			logger.Error("Failed to update subscription", "Error", "updateResult is nil")
			return subscription, err
		}

		// Update the local state with the updated subscription
		subscription = updateResult

		// If the subscription is past due now, wait until it is activated again
		// the payment retries are handled by the Dunning workflow
		if subscription.Status == entities.SubscriptionStatusPastDue {
			err = workflow.Await(ctx, func() bool {
				logger.Debug(fmt.Sprintf("Workflow paused until subscription is activated [%s][%s][%s]", subscription.OrgId, subscription.Id, subscription.Status))
				return subscription.IsRunning() ||
					subscription.Status == entities.SubscriptionStatusCancelled
			})
		}

		// the subscription was successfully charged, update the subscription state
		// and prepare for the next billing period
		logger.Info(fmt.Sprintf("[%s][%s] Charging cycle completed [status=%s][renewsAt=%s][cycles=%d][amount=%d]",
			subscription.OrgId,
			subscription.Id,
			subscription.Status,
			subscription.GetNextChargeDate(),
			subscription.CyclesProcessed,
			subscription.Amount))
	}
	logger.Info(fmt.Sprintf("Completed %s, Total Charged: %d", workflow.GetInfo(ctx).WorkflowExecution.ID, subscription.TotalRevenue))
	return subscription, nil
}
