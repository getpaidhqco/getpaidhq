package workflows

import (
	"payloop/internal/adapter/hatchet/steps"
	"payloop/internal/core/port"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewPaymentRefundedWorkflow handles a single refund event. One step, retried
// indefinitely on failure (mirrors the Temporal version's lack of a max-attempts).
func NewPaymentRefundedWorkflow(client *hatchet.Client, orderSteps *steps.OrderSteps) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("payment-refunded",
		func(ctx hatchet.Context, input PaymentRefundedInput) (port.WorkflowResult, error) {
			return orderSteps.HandlePaymentRefundedEvent(ctx, input.PaymentContext)
		},
		hatchet.WithExecutionTimeout(10*time.Second),
		hatchet.WithRetries(10),
		hatchet.WithRetryBackoff(1.0, 60),
	)
}
