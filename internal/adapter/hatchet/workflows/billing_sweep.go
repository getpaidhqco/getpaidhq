package workflows

import (
	"time"

	"getpaidhq/internal/core/port"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewBillingSweepWorkflow builds the hourly cron entrypoint. It does NO
// subscription work itself: it lists org ids and spawns one org-billing run
// per tenant (the tenant is the sharding axis — a whale org can't block
// others). Modeled on Lago's SubscriptionsBillerJob. Non-durable: a fresh run
// each tick, plain time.Now() is fine (no replay).
func NewBillingSweepWorkflow(client *hatchet.Client, orgRepo port.OrgRepository, logger port.Logger) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("billing-sweep",
		func(ctx hatchet.Context, _ struct{}) (struct{}, error) {
			ids, err := orgRepo.ListIds(ctx)
			if err != nil {
				logger.Error("billing-sweep: ListIds failed", "err", err.Error())
				return struct{}{}, err
			}
			bucket := time.Now().UTC().Truncate(time.Hour)
			for _, orgId := range ids {
				if _, err := client.RunNoWait(ctx, "org-billing", OrgBillingInput{OrgId: orgId},
					hatchet.WithRunKey(OrgBillingRunKey(orgId, bucket)),
					hatchet.WithRunMetadata(map[string]string{"orgId": orgId}),
				); err != nil {
					logger.Error("billing-sweep: spawn org-billing failed", "orgId", orgId, "err", err.Error())
					// continue: one org's failure must not stop the rest
				}
			}
			logger.Infof("billing-sweep fanned out to %d orgs", len(ids))
			return struct{}{}, nil
		},
		hatchet.WithCron("10 * * * *"), // hourly at :10, mirrors Lago's bill_customers cadence
	)
}

// NewOrgBillingWorkflow builds the per-org fan-out. It does two scans over the
// org's subscriptions each tick: (1) due-for-billing → spawn billing-cycle-runner;
// (2) upcoming renewals → spawn send-renewal-reminder per configured offset stage.
// Both idempotent via run keys. Non-durable: plain time.Now() is fine.
// The reminder policy is resolved PER TENANT (reminderResolver, backed by the
// settings table); a disabled or offset-less config ⇒ no reminders.
func NewOrgBillingWorkflow(client *hatchet.Client, subRepo port.SubscriptionRepository, reminderResolver port.ReminderConfigResolver, logger port.Logger) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("org-billing",
		func(ctx hatchet.Context, in OrgBillingInput) (struct{}, error) {
			now := time.Now().UTC()

			// --- (1) Billing fan-out ---
			due, err := subRepo.FindDueForBilling(ctx, in.OrgId, now)
			if err != nil {
				logger.Error("org-billing: FindDueForBilling failed", "orgId", in.OrgId, "err", err.Error())
				return struct{}{}, err
			}
			for _, sub := range due {
				if _, err := client.RunNoWait(ctx, "billing-cycle-runner", sub,
					hatchet.WithRunKey(BillingRunKey(sub.OrgId, sub.Id, sub.CyclesProcessed)),
					hatchet.WithRunMetadata(map[string]string{"orgId": sub.OrgId, "subscriptionId": sub.Id}),
				); err != nil {
					logger.Error("org-billing: spawn billing-cycle-runner failed",
						"orgId", sub.OrgId, "subscriptionId", sub.Id, "err", err.Error())
				}
			}

			// --- (2) Reminder fan-out (per-tenant config; default fallback) ---
			cfg, err := reminderResolver.ResolveReminderConfig(ctx, in.OrgId)
			if err != nil {
				logger.Error("org-billing: ResolveReminderConfig failed", "orgId", in.OrgId, "err", err.Error())
			} else if cfg.Enabled && len(cfg.Offsets) > 0 {
				maxOffset := cfg.Offsets[0]
				for _, d := range cfg.Offsets {
					if d > maxOffset {
						maxOffset = d
					}
				}
				upcoming, err := subRepo.FindUpcomingRenewals(ctx, in.OrgId, now, maxOffset)
				if err != nil {
					logger.Error("org-billing: FindUpcomingRenewals failed", "orgId", in.OrgId, "err", err.Error())
				} else {
					for _, sub := range upcoming {
						for _, offset := range cfg.Offsets {
							// Stage is active once we've crossed (renews_at - offset).
							// Re-spawning every tick is fine: the run key dedups to one send.
							if now.Before(sub.RenewsAt.Add(-offset)) {
								continue
							}
							if _, err := client.RunNoWait(ctx, "send-renewal-reminder", RenewalReminderInput{Subscription: sub},
								hatchet.WithRunKey(ReminderStageRunKey(sub.OrgId, sub.Id, sub.CyclesProcessed, offset)),
								hatchet.WithRunMetadata(map[string]string{
									"orgId": sub.OrgId, "subscriptionId": sub.Id, "reminderOffset": offset.String(),
								}),
							); err != nil {
								logger.Error("org-billing: spawn send-renewal-reminder failed",
									"orgId", sub.OrgId, "subscriptionId", sub.Id, "err", err.Error())
							}
						}
					}
				}
			}

			if len(due) > 0 {
				logger.Infof("org-billing[%s] spawned %d billing-cycle-runner(s)", in.OrgId, len(due))
			}
			return struct{}{}, nil
		},
	)
}
