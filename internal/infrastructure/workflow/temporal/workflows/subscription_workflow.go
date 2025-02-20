package workflows

import (
	"fmt"
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
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
	logger := workflow.GetLogger(ctx)
	subscription := input

	logger.Info("SubscriptionWorkflow started", "Subscription:", subscription.Id)
	var a *activities.OrderActivities
	// Register query handler for subscription details
	err := workflow.SetQueryHandler(ctx, "get-state", func() (entities.Subscription, error) {
		return subscription, nil
	})
	if err != nil {
		return subscription, err
	}

	handler := func(ctx workflow.Context, newSub entities.Subscription) (entities.Subscription, error) {
		// 👉 update the subscription state
		var prevSub entities.Subscription
		prevSub, subscription = subscription, newSub
		return prevSub, nil
	}

	err = workflow.SetUpdateHandler(ctx, "subscription.paused", handler)
	err = workflow.SetUpdateHandler(ctx, "subscription.cancelled", handler)
	err = workflow.SetUpdateHandler(ctx, "subscription.resumed", handler)
	err = workflow.SetUpdateHandler(ctx, "subscription.activated", handler)

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
		logger.Info(fmt.Sprintf("[%s][%s] blocking until nextBillingDate=[%s]", subscription.OrgId, subscription.Id, subscription.RenewsAt))

		// Calculate the duration until the next billing date
		// Remember to use workflow.Now(ctx) to get the current time
		if subscription.RenewsAt.IsZero() {
			logger.Info("Subscription has no next billing date, ending workflow...")
			// TODO report this error
			break
		}
		duration := subscription.RenewsAt.Sub(workflow.Now(ctx))
		ok, err := workflow.AwaitWithTimeout(ctx, duration, func() bool {
			rollover := workflow.GetInfo(ctx).GetContinueAsNewSuggested()
			return subscription.Status == entities.SubscriptionStatusPaused ||
				subscription.Status == entities.SubscriptionStatusCancelled ||
				rollover
		})
		if err != nil {
			logger.Error("cancellation received", "Error", err.Error(), "status", subscription.Status)
		}
		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			logger.Info("--- ContinueAsNewSuggested", "status", subscription.Status, "size", workflow.GetInfo(ctx).GetCurrentHistorySize())
			return subscription, workflow.NewContinueAsNewError(ctx, SubscriptionWorkflow, subscription)
		}
		if !ok {
			logger.Info(fmt.Sprintf("[%s][%s] Next billing date reached [%s]", subscription.OrgId, subscription.Id, subscription.RenewsAt))
		}

		// If the subscription was paused, wait until it is activated again
		if subscription.Status == entities.SubscriptionStatusPaused {
			err = workflow.Await(ctx, func() bool {
				logger.Debug(fmt.Sprintf("[%s][%s] pause clause", subscription.OrgId, subscription.Id))
				return subscription.Status == entities.SubscriptionStatusActive ||
					subscription.Status == entities.SubscriptionStatusTrial ||
					subscription.Status == entities.SubscriptionStatusCancelled
			})
			continue
		}

		if subscription.Status == entities.SubscriptionStatusPastDue {
			err = workflow.Await(ctx, func() bool {
				logger.Debug("Past due clause", "subscription.Status", subscription.Status)

				// wait until the subscription is moved out of past due status. If active, the renew date
				// must be in the future
				return subscription.Status != entities.SubscriptionStatusPastDue
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

		// Check if the subscription is active
		activeOrTrial := subscription.Status == entities.SubscriptionStatusActive ||
			subscription.Status == entities.SubscriptionStatusTrial ||
			subscription.Status == entities.SubscriptionStatusRetry
		if !activeOrTrial {
			logger.Info("Subscription is not active, skipping billing cycle", "status", subscription.Status)
			continue
		}

		// Double-check the next billing date, it must be in the past
		// E.g. if a paused subscription is activated, the next billing date may be in the future
		if subscription.RenewsAt.After(workflow.Now(ctx)) {
			logger.Info("-------------- why am I here? --------------")
			continue
		}

		logger.Info("Subscription is active, processing cycle")
		// Charge the customer
		var chargeResult payments.ChargeResult
		chargeCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Second * 15,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Minute * 2,
				BackoffCoefficient: 1.0,
			},
		})
		err = workflow.ExecuteActivity(chargeCtx, a.ChargeCustomerForBillingPeriod, subscription).
			Get(chargeCtx, &chargeResult)
		if err != nil {
			// a system error occurred when attempting to charge the customer
			// This shouldn't be reached because of the retry policy which will retry forever
			// The workflow can be stopped using updates

			return subscription, err
		}

		// The charge process ended successfully
		// Update the subscription with the charge result
		var updateResult entities.Subscription
		updateCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 15 * time.Second,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Minute * 10,
				BackoffCoefficient: 1.0,
			},
		})
		err = workflow.ExecuteActivity(updateCtx, a.HandleChargeResult, subscription, chargeResult).
			Get(updateCtx, &updateResult)
		if err != nil {
			logger.Error("Failed to HandleChargeResult", "Error", err.Error())
			return subscription, err
		}
		if updateResult.Id == "" {
			logger.Error("Failed to update subscription", "Error", "updateResult is nil")
			return subscription, err
		}

		// Update the local state with the updated subscription
		subscription = updateResult

		// check the status of the subscription after the charge and update the workflow state
		// TODO this is where the dunning flow might happen

		// the subscription was successfully charged, update the subscription state
		// and prepare for the next billing period
		logger.Info(fmt.Sprintf("[%s][%s] Charging cycle completed [status=%s][renewsAt=%s][cycles=%d][amount=%d]",
			subscription.OrgId,
			subscription.Id,
			subscription.Status,
			subscription.RenewsAt,
			subscription.CyclesProcessed,
			subscription.Amount))
	}
	logger.Info(fmt.Sprintf("Completed %s, Total Charged: %d", workflow.GetInfo(ctx).WorkflowExecution.ID, subscription.TotalRevenue))
	return subscription, nil
}
