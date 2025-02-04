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
	Customer     entities.Customer
	Subscription entities.Subscription
}

// SubscriptionWorkflow is a Temporal workflow that manages a subscription instance
// https://community.temporal.io/t/best-way-to-design-a-subscription-workflow/12047
// https://learn.temporal.io/tutorials/go/build-an-email-drip-campaign/
// https://learn.temporal.io/tutorials/typescript/recurring-billing-system/

func SubscriptionWorkflow(ctx workflow.Context, input SubscriptionInput) (string, error) {
	logger := workflow.GetLogger(ctx)
	subscription := input.Subscription
	//customer := input.Customer

	subscriptionCancelled := false

	logger.Info("SubscriptionWorkflow started", "Subscription:", subscription.Id)
	var a *activities.OrderActivities
	// Register query handler for subscription details
	err := workflow.SetQueryHandler(ctx, "getSubscriptionDetails", func() (entities.Subscription, error) {
		return subscription, nil
	})
	if err != nil {
		return "", err
	}

	// Register update handler for cancelling the subscription
	err = workflow.SetUpdateHandlerWithOptions(ctx, "cancelSubscription", func(_ workflow.Context) error {
		subscriptionCancelled = true
		return nil
	}, workflow.UpdateHandlerOptions{
		Validator: func() error {
			if subscriptionCancelled {
				return fmt.Errorf("Subscription is already cancelled")
			}
			return nil
		},
	})
	if err != nil {
		logger.Error("Failed to register update handler", "Error", err)
		return "", err
	}

	for {
		nextBillingDate := subscription.NextBillingDate()
		logger.Info("Blocking until cancelled or nextBillingDate", "date", nextBillingDate.Format(time.RFC3339))

		// Calculate the duration until the next billing date
		// Remember to use workflow.Now(ctx) to get the current time
		duration := nextBillingDate.Sub(workflow.Now(ctx))
		ok, err := workflow.AwaitWithTimeout(ctx, duration, func() bool {
			return subscriptionCancelled
		})
		if err != nil {
			logger.Error("cancellation received", "Error", err.Error())
		}
		if !ok {
			logger.Info("Blocking await timed out")
		}

		// The wait is over, check if the subscription was cancelled and if not, charge the customer and
		// update local state for the next billing period
		if subscriptionCancelled {
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
