package workflows

import (
	"fmt"
	"slices"
	"time"

	"getpaidhq/internal/core/port"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// SweepCadence normalizes the configured sweep interval (BILLING_SWEEP_INTERVAL)
// to whole minutes clamped to [1m, 60m] — cron granularity is one minute — and
// returns the matching cron expression. The same normalized interval must drive
// the OrgBillingRunKey bucket, or dedup and cadence drift apart.
func SweepCadence(interval time.Duration) (time.Duration, string) {
	m := int(interval.Round(time.Minute).Minutes())
	if m < 1 {
		m = 1
	}
	if m >= 60 {
		return time.Hour, "0 * * * *"
	}
	return time.Duration(m) * time.Minute, fmt.Sprintf("*/%d * * * *", m)
}

// NewBillingSweepWorkflow builds the cron sweep entrypoint (interval from
// BILLING_SWEEP_INTERVAL, default 5m). It does NO
// subscription work itself: it lists org ids and spawns one org-billing run
// per tenant (the tenant is the sharding axis — a whale org can't block
// others). Modeled on Lago's SubscriptionsBillerJob. Non-durable: a fresh run
// each tick, plain time.Now() is fine (no replay).
func NewBillingSweepWorkflow(client *hatchet.Client, orgRepo port.OrgRepository, interval time.Duration, logger port.Logger) *hatchet.StandaloneTask {
	tick, cron := SweepCadence(interval)
	return client.NewStandaloneTask("billing-sweep",
		func(ctx hatchet.Context, _ struct{}) (struct{}, error) {
			ids, err := orgRepo.ListIds(ctx)
			if err != nil {
				logger.Error("billing-sweep: ListIds failed", "err", err.Error())
				return struct{}{}, err
			}
			bucket := time.Now().UTC().Truncate(tick)
			for _, orgId := range ids {
				if _, err := client.RunNoWait(ctx, "org-billing", OrgBillingInput{OrgId: orgId},
					hatchet.WithRunKey(OrgBillingRunKey(orgId, bucket)),
					hatchet.WithRunMetadata(map[string]string{"orgId": orgId}),
				); err != nil {
					logger.Error("billing-sweep: spawn org-billing failed", "orgId", orgId, "err", err.Error())
					// continue: one org's failure must not stop the rest
				}
			}
			logger.Info("billing-sweep fanned out", "orgCount", len(ids))
			return struct{}{}, nil
		},
		// WithWorkflowCron (a WorkflowOption), NOT WithCron (a TaskOption): on a
		// standalone task, WithCron's value lands in taskConfig.onCron and is
		// silently dropped — only WithWorkflowCron populates the registered
		// workflow's OnCron/CronTriggers.
		hatchet.WithWorkflowCron(cron),
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
				maxOffset := slices.Max(cfg.Offsets) // guarded by len(Offsets) > 0 above
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
					if len(upcoming) > 0 {
						logger.Info("org-billing reminder scan", "orgId", in.OrgId, "upcoming", len(upcoming))
					}
				}
			}

			if len(due) > 0 {
				logger.Info("org-billing billing fan-out", "orgId", in.OrgId, "billingSpawned", len(due))
			}
			return struct{}{}, nil
		},
	)
}
