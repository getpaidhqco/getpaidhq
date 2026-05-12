package workflows

import (
	"fmt"
	"payloop/internal/adapter/hatchet/steps"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewPaymentSuccessWorkflow builds the three-step DAG that completes an order
// on a successful payment webhook:
//
//  1. complete-order:        Mark the order paid and capture its row.
//  2. get-subscriptions:     Load any subscriptions tied to the order.
//  3. spawn-subscription-runner:
//                            Start the long-running subscription-runner durable
//                            task via the engine port — same code path as the
//                            HTTP /orders/:id/complete handler.
//
// As in the Temporal version, only the first subscription is processed —
// preserving today's behaviour intentionally.
func NewPaymentSuccessWorkflow(client *hatchet.Client, orderSteps *steps.OrderSteps, engine port.Engine) *hatchet.Workflow {
	wf := client.NewWorkflow("payment-success")

	completeOrder := wf.NewTask("complete-order",
		func(ctx hatchet.Context, input PaymentSuccessInput) (port.WorkflowResult, error) {
			return orderSteps.CompleteOrder(ctx, input.PaymentContext)
		},
		hatchet.WithExecutionTimeout(10*time.Second),
		hatchet.WithRetries(10),
		hatchet.WithRetryBackoff(1.0, 60),
	)

	getSubscriptions := wf.NewTask("get-subscriptions",
		func(ctx hatchet.Context, input PaymentSuccessInput) ([]domain.Subscription, error) {
			return orderSteps.GetOrderSubscriptions(ctx, input.PaymentContext.OrgId, input.PaymentContext.OrderId)
		},
		hatchet.WithParents(completeOrder),
		hatchet.WithExecutionTimeout(60*time.Second),
		hatchet.WithRetries(10),
		hatchet.WithRetryBackoff(1.0, 60),
	)

	wf.NewTask("spawn-subscription-runner",
		func(ctx hatchet.Context, input PaymentSuccessInput) (port.WorkflowResult, error) {
			var subs []domain.Subscription
			if err := ctx.ParentOutput(getSubscriptions, &subs); err != nil {
				return port.WorkflowResult{}, fmt.Errorf("get parent output: %w", err)
			}
			if len(subs) == 0 {
				return port.WorkflowResult{Success: true, Message: "no subscriptions for order"}, nil
			}
			sub := subs[0]

			if err := engine.StartSubscriptionWorkflow(ctx, sub); err != nil {
				return port.WorkflowResult{}, fmt.Errorf("start subscription workflow: %w", err)
			}

			return port.WorkflowResult{Success: true, Message: "spawned subscription-runner"}, nil
		},
		hatchet.WithParents(getSubscriptions),
		hatchet.WithExecutionTimeout(30*time.Second),
		hatchet.WithRetries(5),
		hatchet.WithRetryBackoff(1.0, 30),
	)

	return wf
}
