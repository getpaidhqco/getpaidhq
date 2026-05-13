package workflows

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewBillingCycleWorkflow builds a strictly synchronous one-step DAG that
// runs a single charge. The subscription runner spawns this and inspects the
// returned ChargeResult; the wait-for-webhook + handle-result steps live in
// the runner (DAG steps cannot use DurableContext).
//
// Long backoff with effectively unlimited retries while transient gateway
// errors persist.
func NewBillingCycleWorkflow(client *hatchet.Client, subscriptionService port.SubscriptionService) *hatchet.Workflow {
	wf := client.NewWorkflow("billing-cycle")

	wf.NewTask("charge-customer",
		func(ctx hatchet.Context, input BillingCycleInput) (domain.ChargeResult, error) {
			return subscriptionService.ChargeForBillingPeriod(ctx, input.Subscription)
		},
		hatchet.WithExecutionTimeout(60*time.Second),
		hatchet.WithRetries(50),
		hatchet.WithRetryBackoff(1.2, 600),
	)

	return wf
}
