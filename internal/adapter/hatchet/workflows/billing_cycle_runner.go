package workflows

import (
	"strconv"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewBillingCycleRunnerWorkflow builds a BOUNDED, one-shot durable task that
// processes exactly one billing cycle and exits — unlike the retired immortal
// subscription-runner. Because it always completes, its durable-event-log rows
// live briefly in their birth-date partition and are dropped cleanly by
// retention. See docs/internal/durable-runner-timeouts.md.
//
// Flow (mirrors the old runner's per-cycle body):
//  1. Run the billing-cycle charge DAG (idempotent via billing_<org>_<sub>_<cycle>).
//  2. If the charge is Pending, wait up to 1h for the PSP webhook event.
//  3. Hand the final ChargeResult to the subscription service, which advances
//     state (RenewsAt/CyclesProcessed) or opens dunning.
//
// The ≤1h wait needs eviction (TTL < execution timeout) so it isn't reaped by
// the 5-minute default execution timeout. All side effects must be idempotent
// (see plan Task 0): eviction/restart replays this function from the top.
func NewBillingCycleRunnerWorkflow(client *hatchet.Client, subscriptionService port.SubscriptionService) *hatchet.StandaloneTask {
	return client.NewStandaloneDurableTask("billing-cycle-runner",
		func(ctx hatchet.DurableContext, sub domain.Subscription) (domain.Subscription, error) {
			// 1. Charge (child DAG; durable + idempotent by run key → replay-safe).
			billingRes, err := client.Run(ctx, "billing-cycle", BillingCycleInput{Subscription: sub},
				hatchet.WithRunKey(BillingRunKey(sub.OrgId, sub.Id, sub.CyclesProcessed)),
				hatchet.WithRunMetadata(map[string]string{
					"orgId":          sub.OrgId,
					"subscriptionId": sub.Id,
					"cycle":          strconv.Itoa(sub.CyclesProcessed),
				}),
			)
			if err != nil {
				// Infra failure (e.g. no gateway). Non-fatal: the error is surfaced
				// on the run and we exit; the next hourly sweep re-selects this sub
				// (still due) and retries.
				return sub, err
			}

			var chargeResult domain.ChargeResult
			if err := billingRes.TaskOutput("charge-customer").Into(&chargeResult); err != nil {
				return sub, err
			}

			// 2. Pending → wait up to 1h for the webhook to deliver the final status.
			if chargeResult.Status == domain.PaymentStatusPending {
				webhookKey := WebhookEventKey(sub.OrgId, sub.Id)
				wr, err := ctx.WaitFor(hatchet.OrCondition(
					hatchet.SleepCondition(1*time.Hour),
					hatchet.UserEventCondition(webhookKey, ""),
				))
				if err == nil && containsKey(waitedKeys(wr), webhookKey) {
					var fromWebhook domain.ChargeResult
					if err := unmarshalWaited(wr, webhookKey, &fromWebhook); err == nil {
						chargeResult = fromWebhook
					}
				}
			}

			// 3. Apply the result (idempotent per cycle — see plan Task 0).
			input := port.SubscriptionChargeInput{Subscription: sub, ChargeResult: chargeResult}
			if chargeResult.Status == domain.PaymentStatusSucceeded {
				return subscriptionService.HandleSubscriptionChargeSuccess(ctx, input)
			}
			return subscriptionService.HandleSubscriptionChargeFailure(ctx, input)
		},
		hatchet.WithExecutionTimeout(5*time.Minute), // > eviction TTL
		hatchet.WithEvictionPolicy(&hatchet.EvictionPolicy{
			TTL:                   30 * time.Second, // evict during the ≤1h webhook wait
			AllowCapacityEviction: true,
		}),
	)
}
