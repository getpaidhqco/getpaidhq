package workflows

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"strconv"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewSubscriptionRunnerWorkflow builds the per-subscription long-running
// durable task.
//
// On each iteration:
//
//  1. Compute the next charge date from current state.
//  2. Spawn a charge-reminder durable task one minute before, detached.
//  3. Wait for the charge time OR any of:
//     - update:subscription.paused / .resumed / .cancelled / .activated
//     - update:refresh-state
//     - cancel:<sub>
//     If an event fires, the wait result carries the latest Subscription
//     payload and the loop restarts with that state.
//  4. When the sleep wins, spawn the billing-cycle DAG and await ChargeResult.
//  5. If ChargeResult.Status is Pending, wait up to 1h for a webhook event.
//  6. Hand the (possibly webhook-updated) ChargeResult to SubscriptionService.
//  7. Loop.
//
// Terminal exit: Cancelled, Expired, or a cancel:<sub> event.
func NewSubscriptionRunnerWorkflow(client *hatchet.Client, subscriptionService port.SubscriptionService) *hatchet.StandaloneTask {
	return client.NewStandaloneDurableTask("subscription-runner",
		func(ctx hatchet.DurableContext, sub domain.Subscription) (domain.Subscription, error) {
			for {
				if isTerminalStatus(sub.Status) {
					return sub, nil
				}

				next := sub.GetNextChargeDate()
				if next.IsZero() {
					return sub, nil
				}

				reminderAt := next.Add(-1 * time.Minute)
				_, _ = client.RunNoWait(ctx, "subscription-charge-reminder", ReminderInput{
					Subscription: sub,
					ReminderAt:   reminderAt,
				}, hatchet.WithRunKey(ReminderRunKey(sub.OrgId, sub.Id, reminderAt)),
					hatchet.WithRunMetadata(map[string]string{
						"orgId":          sub.OrgId,
						"subscriptionId": sub.Id,
					}))

				// Wait for the next charge time OR any update / cancel event.
				now, err := ctx.Now()
				if err != nil {
					return sub, err
				}
				wait := next.Sub(now)
				if wait < time.Second {
					wait = time.Second
				}

				pausedKey := UpdateEventKey("subscription.paused", sub.OrgId, sub.Id)
				resumedKey := UpdateEventKey("subscription.resumed", sub.OrgId, sub.Id)
				cancelledKey := UpdateEventKey("subscription.cancelled", sub.OrgId, sub.Id)
				activatedKey := UpdateEventKey("subscription.activated", sub.OrgId, sub.Id)
				refreshKey := UpdateEventKey("refresh-state", sub.OrgId, sub.Id)
				cancelKey := CancelEventKey(sub.OrgId, sub.Id)

				waitResult, err := ctx.WaitFor(hatchet.OrCondition(
					hatchet.SleepCondition(wait),
					hatchet.UserEventCondition(pausedKey, ""),
					hatchet.UserEventCondition(resumedKey, ""),
					hatchet.UserEventCondition(cancelledKey, ""),
					hatchet.UserEventCondition(activatedKey, ""),
					hatchet.UserEventCondition(refreshKey, ""),
					hatchet.UserEventCondition(cancelKey, ""),
				))
				if err != nil {
					return sub, err
				}

				keysSeen := waitedKeys(waitResult)
				if containsKey(keysSeen, cancelKey) {
					return sub, nil
				}

				eventFired := false
				for _, k := range []string{pausedKey, resumedKey, cancelledKey, activatedKey, refreshKey} {
					if containsKey(keysSeen, k) {
						var updated domain.Subscription
						if err := unmarshalWaited(waitResult, k, &updated); err == nil && updated.Id != "" {
							sub = updated
						}
						eventFired = true
						break
					}
				}
				if eventFired {
					continue
				}

				// Sleep won — charge time reached.
				if sub.Status == domain.SubscriptionStatusPaused ||
					sub.Status == domain.SubscriptionStatusNonRenewing {
					continue
				}
				if isTerminalStatus(sub.Status) {
					return sub, nil
				}
				if !sub.IsRunning() {
					continue
				}

				// Double-check the charge date hasn't moved into the future
				// (e.g., a paused subscription was activated mid-loop).
				now2, err := ctx.Now()
				if err != nil {
					return sub, err
				}
				if next.After(now2) {
					continue
				}

				// Billing — child DAG.
				billingRes, err := client.Run(ctx, "billing-cycle", BillingCycleInput{Subscription: sub},
					hatchet.WithRunKey(BillingRunKey(sub.OrgId, sub.Id, sub.CyclesProcessed)),
					hatchet.WithRunMetadata(map[string]string{
						"orgId":          sub.OrgId,
						"subscriptionId": sub.Id,
						"cycle":          strconv.Itoa(sub.CyclesProcessed),
					}))
				if err != nil {
					return sub, err
				}

				var chargeResult domain.ChargeResult
				if err := billingRes.TaskOutput("charge-customer").Into(&chargeResult); err != nil {
					return sub, err
				}

				// On a Pending charge, wait for the webhook to deliver the final status.
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

				chargeInput := domain.SubscriptionChargeInput{Subscription: sub, ChargeResult: chargeResult}
				var updated domain.Subscription
				if chargeResult.Status == domain.PaymentStatusSucceeded {
					updated, err = subscriptionService.HandleSubscriptionChargeSuccess(ctx, chargeInput)
				} else {
					updated, err = subscriptionService.HandleSubscriptionChargeFailure(ctx, chargeInput)
				}
				if err != nil {
					return sub, err
				}
				if updated.Id != "" {
					sub = updated
				}
			}
		},
	)
}

func isTerminalStatus(s domain.SubscriptionStatus) bool {
	return s == domain.SubscriptionStatusCancelled ||
		s == domain.SubscriptionStatusExpired ||
		s == domain.SubscriptionStatusCompleted
}
