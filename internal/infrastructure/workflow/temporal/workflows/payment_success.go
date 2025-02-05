package workflows

import (
	"errors"
	"fmt"
	"go.temporal.io/api/enums/v1"
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/workflow"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"

	temporal "go.temporal.io/sdk/workflow"
)

// Execute executes tasks for processing a successful payment
func PaymentSuccessWorkflow(ctx temporal.Context, payload workflow.WorkflowPayload) (workflow.Result, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("PaymentSuccessWorkflow started")

	// parse the data to make sure we have what we need
	wfData, err := payment_providers.ParsePaymentWebhookContext(payload.Data)
	if err != nil {
		logger.Error("Invalid payload data", "err", err.Error())
		return workflow.Result{}, errors.New("invalid payload data, expected payment_providers.PaymentWebhookContext ")
	}

	var a *activities.OrderActivities

	// ACTIVITY
	// Complete the Order
	var completeOrderResult workflow.Result
	ctx1 := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 10000 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
		},
	})
	err = temporal.ExecuteActivity(ctx1, a.CompleteOrder, workflow.CompleteOrderStepInput{
		PaymentContext: wfData,
	}).Get(ctx1, &completeOrderResult)
	if err != nil {
		logger.Error("[Complete Order] failed with error: ", "Error", err.Error())
		return workflow.Result{}, temporalio.NewApplicationError("Complete Order failed", "", err)
	}

	// ACTIVITY
	// Prepare the subscriptions for the order
	// TODO prepare the subscriptions for the order, returning a list of subscriptions
	var subscriptions []entities.Subscription
	err = temporal.ExecuteActivity(ctx1, a.GetOrderSubscriptions, wfData.OrgId, wfData.OrderId).
		Get(ctx1, &subscriptions)

	// step 2, process the subscriptions
	subscription := subscriptions[0]
	logger.Info("Spawiing child workflow for subscription", "subscription", subscription.Id)
	childWorkflowOptions := temporal.ChildWorkflowOptions{
		WorkflowID:        fmt.Sprintf(`subwf_%s_%s`, subscription.OrgId, subscription.Id),
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
	}
	childCtx := temporal.WithChildOptions(ctx, childWorkflowOptions)
	childWorkflowFuture := temporal.ExecuteChildWorkflow(childCtx, SubscriptionWorkflow, SubscriptionInput{
		Customer:     entities.Customer{},
		Subscription: subscription,
	})

	// Wait for the Child Workflow Execution to spawn
	var childWE temporal.Execution
	if err := childWorkflowFuture.GetChildWorkflowExecution().Get(ctx, &childWE); err != nil {
		logger.Error("Unable to start subscription workflow.", "err", err.Error())
		// update the subscription so that we can retry

		return workflow.Result{
			Success: false,
			Message: "Can't spawn child workflow",
			Payload: completeOrderResult.Payload,
		}, err
	}

	// ACTIVITY
	// store the child workflow execution details against the subscription
	err = temporal.ExecuteActivity(ctx1, a.StoreSubscriptionWorkflowContext, activities.StoreSubscriptionWorkflowContextInput{
		OrgId:          wfData.OrgId,
		SubscriptionId: subscription.Id,
		Execution:      childWE,
	}).Get(ctx1, nil)

	logger.Info("Workflow completed.")
	return workflow.Result{
		Success: true,
		Message: "PaymentSuccessWorkflow completed",
		Payload: completeOrderResult.Payload,
	}, nil
}
