package workflows

import (
	"fmt"
	"go.temporal.io/api/enums/v1"
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/adapter/temporal/activities"
	"payloop/internal/core/domain"
	"time"

	"go.temporal.io/sdk/workflow"
)

type SubscriptionInput struct {
	domain.Subscription `json:"subscription"`
}

// SubscriptionWorkflow is a Temporal workflow that manages a subscription instance
// https://community.temporal.io/t/best-way-to-design-a-subscription-workflow/12047
// https://learn.temporal.io/tutorials/go/build-an-email-drip-campaign/
// https://learn.temporal.io/tutorials/typescript/recurring-billing-system/

func SubscriptionWorkflow(ctx workflow.Context, input domain.Subscription) (domain.Subscription, error) {
	logger := workflow.GetLogger(ctx)
	subscription := input
	// A flag to indicate if the workflow should be refreshed. Used by the "force-update" signal
	restartBillingWait := false

	logger.Info("SubscriptionWorkflow started", "subscriptionId", subscription.Id)
	var a *activities.OrderActivities
	// Register query handler for subscription details
	err := workflow.SetQueryHandler(ctx, "get-state", func() (domain.Subscription, error) {
		return subscription, nil
	})
	if err != nil {
		return subscription, err
	}

	handler := func(ctx workflow.Context, newSub domain.Subscription) (domain.Subscription, error) {
		// 👉 update the subscription state
		var prevSub domain.Subscription
		prevSub, subscription = subscription, newSub
		return prevSub, nil
	}
	forceUpdateHandler := func(ctx workflow.Context, newSub domain.Subscription) (domain.Subscription, error) {
		logger.Info("subscription force update", "subscriptionId", subscription.Id)
		// 👉 update the subscription state
		// Do some validation here
		var prevSub domain.Subscription
		prevSub, subscription = subscription, newSub
		restartBillingWait = true
		return prevSub, nil
	}

	err = workflow.SetUpdateHandler(ctx, "subscription.paused", handler)
	err = workflow.SetUpdateHandler(ctx, "subscription.cancelled", handler)
	err = workflow.SetUpdateHandler(ctx, "subscription.resumed", handler)
	err = workflow.SetUpdateHandler(ctx, "subscription.activated", handler)
	err = workflow.SetUpdateHandler(ctx, "refresh-state", forceUpdateHandler)

	// Register signal handler for cancelling the subscription
	var signalSubscription domain.Subscription
	pausedChannel := workflow.GetSignalChannel(ctx, "subscription.paused")
	activatedChannel := workflow.GetSignalChannel(ctx, "subscription.activated")
	cancelledChannel := workflow.GetSignalChannel(ctx, "subscription.cancelled")
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			selector := workflow.NewSelector(ctx)
			selector.AddReceive(pausedChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalSubscription)
				subscription = signalSubscription
				logger.Info("subscription paused signal", "subscriptionId", subscription.Id, "status", subscription.Status)
			})
			selector.AddReceive(activatedChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalSubscription)
				subscription = signalSubscription
				logger.Info("subscription activated signal", "subscriptionId", subscription.Id, "status", subscription.Status)
			})
			selector.AddReceive(cancelledChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalSubscription)
				logger.Info("received SubscriptionStatusCancelled signal", "subscriptionId", subscription.Id)
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
			logger.Info("subscription has no next billing date, ending workflow")
			// TODO report this error
			break
		}

		// Wait here until the next billing date or until the subscription is cancelled
		// If the subscription state was updated using the "force-update", then we need to
		// restart the wait so that the new state is taken into account
		// RenewsAt is the date when the subscription will be charged again
		// NextRenewalDate is the date when the subscription will be charged again
		duration := nextCharge.Sub(workflow.Now(ctx))
		reminderDuration := time.Duration(1) * time.Minute
		reminderDate := nextCharge.Add(-reminderDuration)
		logger.Info("blocking until next billing date", "orgId", subscription.OrgId, "subscriptionId", subscription.Id, "nextBillingDate", nextCharge)

		// Start the reminder workflow
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID:        fmt.Sprintf(`sub_reminder_[%s]_[%s]_[%s]`, subscription.OrgId, subscription.Id, reminderDate.Format("20060102")),
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
		})
		childWorkflowFuture := workflow.ExecuteChildWorkflow(childCtx, SubscriptionChargeReminder, subscription, reminderDate)
		// Wait for the Child Workflow Execution to spawn
		var childWE workflow.Execution
		if err := childWorkflowFuture.GetChildWorkflowExecution().
			Get(ctx, &childWE); err != nil {
			logger.Error("unable to start subscription reminder workflow", "error", err)
		}
		logger.Info("sending reminder", "reminderDate", reminderDate)

		ok, err := workflow.AwaitWithTimeout(ctx, duration, func() bool {
			rollover := workflow.GetInfo(ctx).GetContinueAsNewSuggested()
			return subscription.Status == domain.SubscriptionStatusPaused ||
				subscription.Status == domain.SubscriptionStatusCancelled ||
				rollover ||
				restartBillingWait
		})
		if err != nil {
			logger.Error("workflow await was interrupted",
				"error", err,
				"status", string(subscription.Status),
				"nextBillingDate", nextCharge.String(),
				"restartBillingWait", restartBillingWait)
		}
		if restartBillingWait {
			logger.Info("restart billing wait - the subscription state was refreshed, clearing the flag and restarting the loop")
			restartBillingWait = false
			continue
		}
		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			logger.Info("continue as new suggested", "status", subscription.Status, "size", workflow.GetInfo(ctx).GetCurrentHistorySize())
			return subscription, workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, subscription)
		}
		if !ok {
			logger.Info("next billing date reached", "orgId", subscription.OrgId, "subscriptionId", subscription.Id, "nextBillingDate", nextCharge)
		}

		// If the subscription was paused, wait until it is activated again
		if subscription.Status == domain.SubscriptionStatusPaused {
			err = workflow.Await(ctx, func() bool {
				logger.Debug("pause clause", "orgId", subscription.OrgId, "subscriptionId", subscription.Id)
				return subscription.IsRunning() ||
					subscription.Status == domain.SubscriptionStatusCancelled
			})
			continue
		}

		if subscription.Status == domain.SubscriptionStatusNonRenewing {
			err = workflow.Await(ctx, func() bool {
				logger.Debug("past due clause", "status", subscription.Status)

				// wait until the subscription is moved out of the non-renewing state.
				return subscription.Status != domain.SubscriptionStatusNonRenewing
			})
			continue
		}

		// The wait is over, check if the subscription was cancelled and if not, charge the customer and
		// update local state for the next billing period
		if subscription.Status == domain.SubscriptionStatusCancelled {
			logger.Info("subscription is cancelled, ending workflow")
			break
		}
		if subscription.Status == domain.SubscriptionStatusExpired {
			logger.Info("subscription is expired, ending workflow")
			break
		}

		if !subscription.IsRunning() {
			logger.Info("subscription is not in a running state, skipping billing cycle", "status", subscription.Status)
			continue
		}

		// Double-check the next billing date, it must be in the past
		// E.g. if a paused subscription is activated, the next billing date may be in the future
		if nextCharge.After(workflow.Now(ctx)) {
			logger.Info("reached the billing process but renew date is in the future, skipping billing cycle", "nextBillingDate", nextCharge)
			continue
		}

		logger.Info("subscription is active, processing cycle")
		// Charge the customer
		var chargeResult domain.ChargeResult
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
			logger.Error("ChargeCustomerForBillingPeriod failed completely, ending workflow", "error", err)
			return subscription, err
		}

		// Charge is completed
		// If payment status is Pending, then we must wait for a webhook to complete the payment
		if chargeResult.Status == domain.PaymentStatusPending {
			// Wait for the webhook
			selector := workflow.NewSelector(ctx)
			webhookChan := workflow.GetSignalChannel(ctx, "webhook-signal")

			selector.AddReceive(webhookChan, func(c workflow.ReceiveChannel, more bool) {
				logger.Info("received webhook signal")
				c.Receive(ctx, &chargeResult)
			})

			// Wait for either the webhook or a timeout
			timeout := workflow.NewTimer(ctx, 1*time.Hour)
			selector.AddFuture(timeout, func(f workflow.Future) {
				// Handle timeout
				logger.Error("timeout waiting for payment webhook")
			})

			selector.Select(ctx)
		}

		// The charge process ended successfully
		// Update the subscription with the charge result
		var updateResult domain.Subscription
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
			logger.Error("failed to HandleChargeResult", "error", err)
			return subscription, err
		}
		if updateResult.Id == "" {
			logger.Error("failed to update subscription", "error", "updateResult is nil")
			return subscription, err
		}

		// Update the local state with the updated subscription
		subscription = updateResult

		// check the status of the subscription after the charge and update the workflow state
		// TODO this is where the dunning flow might happen

		// the subscription was successfully charged, update the subscription state
		// and prepare for the next billing period
		logger.Info("charging cycle completed",
			"orgId", subscription.OrgId,
			"subscriptionId", subscription.Id,
			"status", subscription.Status,
			"renewsAt", subscription.GetNextChargeDate(),
			"cyclesProcessed", subscription.CyclesProcessed,
			"amount", subscription.Amount)
	}
	logger.Info("completed workflow", "workflowId", workflow.GetInfo(ctx).WorkflowExecution.ID, "totalRevenue", subscription.TotalRevenue)
	return subscription, nil
}
