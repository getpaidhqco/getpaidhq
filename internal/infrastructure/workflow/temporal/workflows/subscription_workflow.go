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

func SubscriptionWorkflow(ctx workflow.Context, input entities.Subscription) (string, error) {
	logger := workflow.GetLogger(ctx)
	subscription := input

	logger.Info("SubscriptionWorkflow started", "Subscription:", subscription.Id)
	var a *activities.OrderActivities
	// Register query handler for subscription details
	err := workflow.SetQueryHandler(ctx, "getSubscriptionDetails", func() (entities.Subscription, error) {
		return subscription, nil
	})
	if err != nil {
		return "", err
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
				subscription.Status = entities.SubscriptionStatusPaused
			})
			selector.AddReceive(activatedChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalSubscription)
				subscription.Status = entities.SubscriptionStatusActive
			})
			selector.AddReceive(cancelledChannel, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalSubscription)
				subscription.Status = entities.SubscriptionStatusCancelled
			})
			selector.Select(ctx)
		}
	})

	for {
		nextBillingDate := subscription.NextBillingDate()
		logger.Info("Blocking until cancelled or nextBillingDate", "date", nextBillingDate.Format(time.RFC3339))

		// Calculate the duration until the next billing date
		// Remember to use workflow.Now(ctx) to get the current time
		duration := nextBillingDate.Sub(workflow.Now(ctx))
		ok, err := workflow.AwaitWithTimeout(ctx, duration, func() bool {
			return subscription.Status == entities.SubscriptionStatusPaused ||
				subscription.Status == entities.SubscriptionStatusCancelled
		})
		if err != nil {
			logger.Error("cancellation received", "Error", err.Error(), "status", subscription.Status)
		}
		if !ok {
			logger.Info("Next billing date reached", "date", nextBillingDate.Format(time.RFC3339))
		}

		if subscription.Status == entities.SubscriptionStatusPaused {
			err = workflow.Await(ctx, func() bool {
				logger.Debug("Pause clause", "subscription.Status", subscription.Status)
				return subscription.Status == entities.SubscriptionStatusActive ||
					subscription.Status == entities.SubscriptionStatusCancelled
			})
		}

		// The wait is over, check if the subscription was cancelled and if not, charge the customer and
		// update local state for the next billing period
		if subscription.Status == entities.SubscriptionStatusCancelled {
			logger.Info("Subscription is cancelled, ending workflow...")
			break
		}

		// Charge the customer
		var chargeResult payments.ChargeResult
		chargeCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 1000 * time.Second,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Minute,
				BackoffCoefficient: 1.0,
			},
		})
		err = workflow.ExecuteActivity(chargeCtx, a.ChargeCustomerForBillingPeriod, subscription).
			Get(chargeCtx, &chargeResult)
		if err != nil {
			logger.Error("Failed to charge customer", "Error", err.Error())
			return "", err
		}

		// Update the subscription with the charge result
		var updateResult entities.Subscription
		updateCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: 10000 * time.Second,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Minute,
				BackoffCoefficient: 1.0,
			},
		})
		err = workflow.ExecuteActivity(updateCtx, a.StoreChargeResults, subscription, chargeResult).
			Get(updateCtx, &updateResult)
		if err != nil {
			logger.Error("Failed to StoreChargeResults", "Error", err.Error())
			return "", err
		}

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
	return "ok", nil
}
