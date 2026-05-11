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
//                            Spawn the long-running subscription-runner durable
//                            task with a deterministic run key, then persist
//                            the run id in the settings table for later lookup.
//
// As in the Temporal version, only the first subscription is processed —
// preserving today's behaviour intentionally.
func NewPaymentSuccessWorkflow(client *hatchet.Client, orderSteps *steps.OrderSteps) *hatchet.Workflow {
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

			ref, err := client.RunNoWait(ctx, "subscription-runner", sub,
				hatchet.WithRunKey(SubscriptionRunKey(sub.OrgId, sub.Id)),
			)
			if err != nil {
				return port.WorkflowResult{}, fmt.Errorf("spawn subscription-runner: %w", err)
			}

			err = orderSteps.StoreSubscriptionWorkflowContext(ctx, steps.StoreSubscriptionWorkflowContextInput{
				OrgId:          sub.OrgId,
				SubscriptionId: sub.Id,
				RunID:          ref.RunId,
				WorkflowName:   "subscription-runner",
			})
			if err != nil {
				return port.WorkflowResult{}, fmt.Errorf("store run id: %w", err)
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
