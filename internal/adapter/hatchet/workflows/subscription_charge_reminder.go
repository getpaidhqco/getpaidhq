package workflows

import (
	"payloop/internal/adapter/hatchet/steps"
	"payloop/internal/core/port"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewSubscriptionChargeReminderWorkflow sleeps until the reminder time
// (durably — survives worker restarts) and then sends the renewal reminder.
// Spawned with RunNoWait and a deterministic run key by the subscription
// runner so the parent never blocks on it.
func NewSubscriptionChargeReminderWorkflow(client *hatchet.Client, orderSteps *steps.OrderSteps) *hatchet.StandaloneTask {
	return client.NewStandaloneDurableTask("subscription-charge-reminder",
		func(ctx hatchet.DurableContext, input ReminderInput) (port.WorkflowResult, error) {
			now, err := ctx.Now()
			if err != nil {
				return port.WorkflowResult{}, err
			}
			wait := input.ReminderAt.Sub(now)
			if wait > 0 {
				if _, err := ctx.SleepFor(wait); err != nil {
					return port.WorkflowResult{}, err
				}
			}
			if err := orderSteps.ProcessReminderEvent(ctx, input.Subscription); err != nil {
				return port.WorkflowResult{Success: false}, err
			}
			return port.WorkflowResult{Success: true, Message: "reminder sent"}, nil
		},
		hatchet.WithExecutionTimeout(10*time.Second),
	)
}
