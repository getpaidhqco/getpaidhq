package workflows

import (
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// RenewalReminderInput carries the subscription whose renewal reminder should
// be sent. (Named distinctly from the old runner's `ReminderInput`, which still
// exists in subscription_charge_reminder.go until Task 7 — avoids a duplicate
// declaration so every task builds green.)
type RenewalReminderInput struct {
	Subscription domain.Subscription `json:"subscription"`
}

// NewSendRenewalReminderWorkflow builds a short, non-durable task that sends one
// renewal reminder. The per-(cycle, offset) run key (ReminderStageRunKey) makes
// each stage fire once per cycle, so this task itself just performs the send.
func NewSendRenewalReminderWorkflow(client *hatchet.Client, subscriptionService port.SubscriptionService) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("send-renewal-reminder",
		func(ctx hatchet.Context, in RenewalReminderInput) (struct{}, error) {
			err := subscriptionService.SendRenewalReminder(ctx, in.Subscription.OrgId, in.Subscription.Id)
			return struct{}{}, err
		},
	)
}
