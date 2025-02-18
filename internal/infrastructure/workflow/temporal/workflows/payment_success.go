package workflows

import (
	"errors"
	"fmt"
	"go.temporal.io/api/enums/v1"
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/infrastructure/workflow/temporal/types"
	"time"

	temporal "go.temporal.io/sdk/workflow"
)

// Execute executes tasks for processing a successful payment
func PaymentSuccessWorkflow(ctx temporal.Context, payload interfaces.WorkflowPayload) (interfaces.Result, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("PaymentSuccessWorkflow started")

	// parse the data to make sure we have what we need
	paymentWebhookContext, err := payment_providers.ParsePaymentWebhookContext(payload.Data)
	if err != nil {
		logger.Error("Invalid payload data", "err", err.Error())
		return interfaces.Result{}, errors.New("invalid payload data, expected payment_providers.PaymentWebhookContext ")
	}

	var a *activities.OrderActivities

	// ACTIVITY
	// Complete the Order
	var completeOrderResult interfaces.Result
	ctx1 := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
		},
	})
	err = temporal.ExecuteActivity(ctx1, a.CompleteOrder, paymentWebhookContext).Get(ctx1, &completeOrderResult)
	if err != nil {
		logger.Error("[Complete Order] failed with error: ", "Error", err.Error())
		return interfaces.Result{}, temporalio.NewApplicationError("Complete Order failed", "", err)
	}

	// ACTIVITY
	// Prepare the subscriptions for the order
	var subscriptions []entities.Subscription
	ctx2 := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 10000 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
		},
	})
	err = temporal.ExecuteActivity(ctx2, a.GetOrderSubscriptions, paymentWebhookContext.OrgId, paymentWebhookContext.OrderId).
		Get(ctx2, &subscriptions)

	// ACTIVITY
	// process the subscriptions
	subscription := subscriptions[0]
	logger.Info("Spawning child workflow for subscription", "subscription", subscription.Id)
	childCtx := temporal.WithChildOptions(ctx, temporal.ChildWorkflowOptions{
		WorkflowID:        fmt.Sprintf(`subscription_[%s]_[%s]`, subscription.OrgId, subscription.Id),
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
	})
	childWorkflowFuture := temporal.ExecuteChildWorkflow(childCtx, SubscriptionWorkflow, subscription)

	// Wait for the Child Workflow Execution to spawn
	var childWE temporal.Execution
	if err := childWorkflowFuture.GetChildWorkflowExecution().
		Get(ctx, &childWE); err != nil {
		logger.Error("Unable to start subscription workflow.", "err", err.Error())
		// update the subscription so that we can retry

		return interfaces.Result{
			Success: false,
			Message: "Can't spawn child workflow",
			Payload: completeOrderResult.Payload,
		}, err
	}

	// ACTIVITY
	// store the child workflow execution details against the subscription
	err = temporal.ExecuteActivity(ctx1, a.StoreSubscriptionWorkflowContext, types.StoreSubscriptionWorkflowContextInput{
		OrgId:          paymentWebhookContext.OrgId,
		SubscriptionId: subscription.Id,
		Execution:      childWE,
	}).Get(ctx1, nil)

	logger.Info("[payment_success] Workflow completed.")
	return interfaces.Result{
		Success: true,
		Message: "PaymentSuccessWorkflow completed",
		Payload: completeOrderResult.Payload,
	}, nil
}
