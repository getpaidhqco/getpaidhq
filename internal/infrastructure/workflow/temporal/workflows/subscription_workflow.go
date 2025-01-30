package workflows

import (
	"fmt"
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/domain/entities"
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

	subscriptionCancelled := false
	totalCharged := 0
	billingPeriodNumber := 0
	billingPeriodChargeAmount := input.Subscription.Amount
	nextBillingDate := input.Subscription.StartDate
	customer := input.Customer

	logger.Info("SubscriptionWorkflow started", "Subscription:", input.Subscription.Id)
	var a *activities.OrderActivities
	// Register query handler for subscription details
	err := workflow.SetQueryHandler(ctx, "getSubscriptionDetails", func() (string, error) {
		return "hello", nil
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
		logger.Info("Blocking until cancelled or nextBillingDate", "date", nextBillingDate.Format(time.RFC3339))

		// Calculate the duration until the next billing date
		// Remember to use workflow.Now(ctx) to get the current time
		// https://medium.com/@sanhdoan/understanding-non-determinism-in-temporal-io-why-it-matters-how-to-avoid-it-3d397d8a5793
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
		options := workflow.ActivityOptions{
			StartToCloseTimeout: 1000 * time.Second,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Minute,
				BackoffCoefficient: 1.0,
			},
		}
		ctx1 := workflow.WithActivityOptions(ctx, options)
		logger.Info("Charging customer", "billingPeriodNumber", billingPeriodNumber, "amount", billingPeriodChargeAmount)
		err = workflow.ExecuteActivity(ctx1, a.ChargeCustomerForBillingPeriod, customer, billingPeriodChargeAmount).
			Get(ctx, nil)
		if err != nil {
			return "", err
		}

		// Update the total charged and prepare for the next billing period
		totalCharged += billingPeriodChargeAmount
		billingPeriodNumber++
		nextBillingDate = workflow.Now(ctx).Add(time.Second * time.Duration(10))

	}
	logger.Info(fmt.Sprintf("Completed %s, Total Charged: %d", workflow.GetInfo(ctx).WorkflowExecution.ID, totalCharged))
	return "ok", nil
}
