package workflows

import (
	"fmt"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewPaymentSuccessWorkflow builds the three-step DAG that completes an order
// on a successful payment webhook:
//
//  1. complete-order:        Mark the order paid and capture the row.
//  2. get-subscriptions:     Load any subscriptions tied to the order.
//  3. start-subscription-lifecycle:
//     Kick off the subscription's billing lifecycle via the
//     engine port — same code path as the HTTP
//     /orders/:id/complete handler. On Hatchet this may spawn
//     an immediate first charge (billing-cycle-runner) when
//     the subscription is already due; otherwise it is a
//     no-op and the cron sweep drives renewals.
//
// Only the first subscription is processed — preserving today's behaviour
// intentionally.
func NewPaymentSuccessWorkflow(
	client *hatchet.Client,
	orderService port.OrderWorkflowService,
	subscriptionRepo port.SubscriptionRepository,
	engine port.Engine,
) *hatchet.Workflow {
	wf := client.NewWorkflow("payment-success")

	completeOrder := wf.NewTask("complete-order",
		func(ctx hatchet.Context, input PaymentSuccessInput) (domain.Order, error) {
			return orderService.CompleteCheckoutSession(ctx, port.CompleteCheckoutSessionInput{
				OrgId:          input.PaymentContext.OrgId,
				OrderId:        input.PaymentContext.OrderId,
				PaymentContext: input.PaymentContext,
			})
		},
		hatchet.WithExecutionTimeout(10*time.Second),
		hatchet.WithRetries(10),
		hatchet.WithRetryBackoff(1.0, 60),
	)

	getSubscriptions := wf.NewTask("get-subscriptions",
		func(ctx hatchet.Context, input PaymentSuccessInput) ([]domain.Subscription, error) {
			return subscriptionRepo.FindByOrderId(ctx, input.PaymentContext.OrgId, input.PaymentContext.OrderId)
		},
		hatchet.WithParents(completeOrder),
		hatchet.WithExecutionTimeout(60*time.Second),
		hatchet.WithRetries(10),
		hatchet.WithRetryBackoff(1.0, 60),
	)

	wf.NewTask("start-subscription-lifecycle",
		func(ctx hatchet.Context, input PaymentSuccessInput) (domain.Subscription, error) {
			var subs []domain.Subscription
			if err := ctx.ParentOutput(getSubscriptions, &subs); err != nil {
				return domain.Subscription{}, fmt.Errorf("get parent output: %w", err)
			}
			if len(subs) == 0 {
				return domain.Subscription{}, nil
			}
			sub := subs[0]

			if err := engine.StartSubscriptionWorkflow(ctx, sub); err != nil {
				return domain.Subscription{}, fmt.Errorf("start subscription workflow: %w", err)
			}

			return sub, nil
		},
		hatchet.WithParents(getSubscriptions),
		hatchet.WithExecutionTimeout(30*time.Second),
		hatchet.WithRetries(5),
		hatchet.WithRetryBackoff(1.0, 30),
	)

	return wf
}
