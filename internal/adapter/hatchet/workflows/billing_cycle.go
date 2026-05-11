package workflows

import (
	"payloop/internal/adapter/hatchet/steps"
	"payloop/internal/core/domain"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewBillingCycleWorkflow builds a strictly synchronous one-step DAG that
// runs a single charge. The subscription runner spawns this and inspects the
// returned ChargeResult; the wait-for-webhook + handle-result steps live in
// the runner (DAG steps cannot use DurableContext).
//
// Mirrors the Temporal workflow's retry shape: long backoff, effectively
// unlimited retries while transient gateway errors persist.
func NewBillingCycleWorkflow(client *hatchet.Client, orderSteps *steps.OrderSteps) *hatchet.Workflow {
	wf := client.NewWorkflow("billing-cycle")

	wf.NewTask("charge-customer",
		func(ctx hatchet.Context, input BillingCycleInput) (domain.ChargeResult, error) {
			return orderSteps.ChargeCustomerForBillingPeriod(ctx, input.Subscription)
		},
		hatchet.WithExecutionTimeout(60*time.Second),
		hatchet.WithRetries(50),
		hatchet.WithRetryBackoff(1.2, 600),
	)

	return wf
}
