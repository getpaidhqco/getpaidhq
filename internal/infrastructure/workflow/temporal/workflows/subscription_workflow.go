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
	err := workflow.SetQueryHandler(ctx, "getSubscriptionDetails", func() (entities.Subscription, error) {
		return subscription, nil
	})
	if err != nil {
		return subscription, err
	}

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
		logger.Info("Blocking until cancelled or nextBillingDate", "date", subscription.RenewsAt.Format(time.RFC3339))

		// Calculate the duration until the next billing date
		// Remember to use workflow.Now(ctx) to get the current time
		duration := subscription.RenewsAt.Sub(workflow.Now(ctx))
		ok, err := workflow.AwaitWithTimeout(ctx, duration, func() bool {
			return subscription.Status == entities.SubscriptionStatusPaused ||
				subscription.Status == entities.SubscriptionStatusCancelled
		})
		if err != nil {
			logger.Error("cancellation received", "Error", err.Error(), "status", subscription.Status)
		}
		if !ok {
			logger.Info("Next billing date reached", "date", subscription.RenewsAt.Format(time.RFC3339))
		}

		// If the subscription was paused, wait until it is activated again
		if subscription.Status == entities.SubscriptionStatusPaused {
			err = workflow.Await(ctx, func() bool {
				logger.Debug("Pause clause", "subscription.Status", subscription.Status)
				return subscription.Status == entities.SubscriptionStatusActive ||
					subscription.Status == entities.SubscriptionStatusTrial ||
					subscription.Status == entities.SubscriptionStatusCancelled
			})
		}

		// The wait is over, check if the subscription was cancelled and if not, charge the customer and
		// update local state for the next billing period
		if subscription.Status == entities.SubscriptionStatusCancelled {
			logger.Info("Subscription is cancelled, ending workflow...")
			break
		}

		// Check if the subscription is active
		activeOrTrial := subscription.Status == entities.SubscriptionStatusActive ||
			subscription.Status == entities.SubscriptionStatusTrial
		if !activeOrTrial {
			logger.Info("Subscription is not active, skipping billing cycle")
			continue
		}

		// Double-check the next billing date, it must be in the past
		// E.g. if a paused subscription is activated, the next billing date may be in the future
		if subscription.RenewsAt.After(workflow.Now(ctx)) {
			logger.Info("Next billing date is in the future, skipping billing cycle")
			continue
		}

		logger.Info("Subscription is active, processing cycle")
		// Charge the customer
		var chargeResult payments.ChargeResult
		chargeCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 60 * time.Second,
			RetryPolicy: &temporalio.RetryPolicy{
				MaximumAttempts:    1,
				InitialInterval:    time.Second * 15,
				BackoffCoefficient: 1.0,
			},
		})
		err = workflow.ExecuteActivity(chargeCtx, a.ChargeCustomerForBillingPeriod, subscription).
			Get(chargeCtx, &chargeResult)
		if err != nil {
			logger.Error("Failed to charge customer", "Error", err.Error())
			// TODO this is where the subscription goes into PAST_DUE status
			return subscription, err
		}

		// Update the subscription with the charge result
		var updateResult entities.Subscription
		updateCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10000 * time.Second,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Second * 15,
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

		// the subscription was successfully charged, update the subscription state
		// and prepare for the next billing period
		logger.Info("Charging cycle completed",
			"orgId", subscription.OrgId,
			"id", subscription.Id,
			"billingPeriodNumber", subscription.CyclesProcessed,
			"amount", subscription.Amount)
	}
	logger.Info(fmt.Sprintf("Completed %s, Total Charged: %d", workflow.GetInfo(ctx).WorkflowExecution.ID, subscription.TotalRevenue))
	return subscription, nil
}
