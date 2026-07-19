package workflows

import (
	"errors"
	"fmt"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	errors2 "getpaidhq/internal/lib/errors"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/worker"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewPaymentSuccessWorkflow builds the three-step DAG that completes an order
// on a successful payment webhook:
//
//  1. complete-order:        Mark the order paid and capture the row.
//  2. get-subscriptions:     Load any subscriptions tied to the order.
//  3. start-subscription-lifecycle:
//     Kick off the billing lifecycle for EVERY subscription on the order via the
//     engine port — same code path as the HTTP /orders/:id/complete handler. An
//     order can carry more than one subscription (one per billing cadence), so all
//     are started; Start*Workflow is idempotent via deterministic ids.
func NewPaymentSuccessWorkflow(
	client *hatchet.Client,
	orderService port.OrderWorkflowService,
	subscriptionRepo port.SubscriptionRepository,
	engine port.Engine,
) *hatchet.Workflow {
	wf := client.NewWorkflow("payment-success")

	completeOrder := wf.NewTask("complete-order",
		func(ctx hatchet.Context, input PaymentSuccessInput) (domain.Order, error) {
			order, err := orderService.CompleteCheckoutSession(ctx, port.CompleteCheckoutSessionInput{
				OrgId:          input.PaymentContext.OrgId,
				OrderId:        input.PaymentContext.OrderId,
				PaymentContext: input.PaymentContext,
			})
			if err != nil && isPermanentCompleteOrderError(err) {
				return domain.Order{}, worker.NewNonRetryableError(err)
			}
			return order, err
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
		func(ctx hatchet.Context, input PaymentSuccessInput) ([]domain.Subscription, error) {
			var subs []domain.Subscription
			if err := ctx.ParentOutput(getSubscriptions, &subs); err != nil {
				return nil, fmt.Errorf("get parent output: %w", err)
			}
			for _, sub := range subs {
				if err := engine.StartSubscriptionWorkflow(ctx, sub); err != nil {
					return nil, fmt.Errorf("start subscription workflow %s: %w", sub.Id, err)
				}
			}
			return subs, nil
		},
		hatchet.WithParents(getSubscriptions),
		hatchet.WithExecutionTimeout(30*time.Second),
		hatchet.WithRetries(5),
		hatchet.WithRetryBackoff(1.0, 30),
	)

	return wf
}

func isPermanentCompleteOrderError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, port.ErrNotFound) {
		return true
	}

	var custom errors2.CustomError
	if errors.As(err, &custom) {
		switch custom.Type {
		case errors2.BadRequestError, errors2.ConflictError, errors2.NotFoundError, errors2.ValidationError:
			return true
		}
	}

	return false
}
