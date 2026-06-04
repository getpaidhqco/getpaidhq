package workflows

import (
	"time"

	"go.temporal.io/api/enums/v1"
	temporalio "go.temporal.io/sdk/temporal"
	temporal "go.temporal.io/sdk/workflow"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// PaymentSuccessWorkflow handles a payment-success webhook. Mirrors
// internal/adapter/hatchet/workflows/payment_success.go:
//
//  1. complete-order:        Mark the order paid and capture the row.
//  2. get-subscriptions:     Load any subscriptions tied to the order.
//  3. spawn-subscription-runner:
//     Start the per-subscription runner as a detached
//     child workflow with a deterministic id.
//
// Only the first subscription is processed (matching Hatchet today).
func PaymentSuccessWorkflow(ctx temporal.Context, input PaymentSuccessInput) (port.WorkflowResult, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("PaymentSuccessWorkflow started", "orderId", input.PaymentContext.OrderId)

	var act *activities.OrderActivities

	completeCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
			MaximumAttempts:    10,
		},
	})
	var order domain.Order
	if err := temporal.ExecuteActivity(completeCtx, act.CompleteOrder, input.PaymentContext).
		Get(completeCtx, &order); err != nil {
		return port.WorkflowResult{}, err
	}

	subsCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
			MaximumAttempts:    10,
		},
	})
	var subs []domain.Subscription
	if err := temporal.ExecuteActivity(subsCtx, act.GetOrderSubscriptions, input.PaymentContext.OrgId, input.PaymentContext.OrderId).
		Get(subsCtx, &subs); err != nil {
		return port.WorkflowResult{}, err
	}
	if len(subs) == 0 {
		return port.WorkflowResult{Success: true, Message: "no subscriptions for order", Payload: order}, nil
	}

	sub := subs[0]
	childCtx := temporal.WithChildOptions(ctx, temporal.ChildWorkflowOptions{
		WorkflowID:            SubscriptionWorkflowID(sub.OrgId, sub.Id),
		ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	})
	if err := temporal.ExecuteChildWorkflow(childCtx, SubscriptionWorkflow, sub).
		GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		logger.Error("Unable to start subscription runner", "err", err.Error())
		return port.WorkflowResult{Success: false, Message: "Can't spawn subscription runner", Payload: order}, err
	}

	logger.Info("PaymentSuccessWorkflow completed", "orderId", order.Id)
	return port.WorkflowResult{Success: true, Message: "PaymentSuccessWorkflow completed", Payload: order}, nil
}
