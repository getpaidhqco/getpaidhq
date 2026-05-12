package workflows

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewPaymentRefundedWorkflow handles a single refund event. One step, retried
// indefinitely on failure (mirrors the Temporal version's lack of a max-attempts).
func NewPaymentRefundedWorkflow(client *hatchet.Client, paymentService port.PaymentService) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("payment-refunded",
		func(ctx hatchet.Context, input PaymentRefundedInput) (domain.Payment, error) {
			return paymentService.ProcessRefund(ctx, input.PaymentContext)
		},
		hatchet.WithExecutionTimeout(10*time.Second),
		hatchet.WithRetries(10),
		hatchet.WithRetryBackoff(1.0, 60),
	)
}
