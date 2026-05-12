package workflows

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewSubscriptionChargeReminderWorkflow sleeps until the reminder time
// (durably — survives worker restarts) and then sends the renewal reminder.
// Spawned with RunNoWait and a deterministic run key by the subscription
// runner so the parent never blocks on it.
func NewSubscriptionChargeReminderWorkflow(client *hatchet.Client, subscriptionService port.SubscriptionService) *hatchet.StandaloneTask {
	return client.NewStandaloneDurableTask("subscription-charge-reminder",
		func(ctx hatchet.DurableContext, input ReminderInput) (domain.Subscription, error) {
			now, err := ctx.Now()
			if err != nil {
				return input.Subscription, err
			}
			wait := input.ReminderAt.Sub(now)
			if wait > 0 {
				if _, err := ctx.SleepFor(wait); err != nil {
					return input.Subscription, err
				}
			}
			if err := subscriptionService.SendRenewalReminder(ctx, input.Subscription.OrgId, input.Subscription.Id); err != nil {
				return input.Subscription, err
			}
			return input.Subscription, nil
		},
		hatchet.WithExecutionTimeout(10*time.Second),
	)
}
