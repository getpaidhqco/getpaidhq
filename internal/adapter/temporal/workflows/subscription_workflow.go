package workflows

import (
	"time"

	"go.temporal.io/api/enums/v1"
	temporalio "go.temporal.io/sdk/temporal"
	temporal "go.temporal.io/sdk/workflow"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
)

// SubscriptionWorkflow is the per-subscription long-running runner. Mirrors
// internal/adapter/hatchet/workflows/subscription_runner.go.
//
// On each iteration:
//
//  1. Compute the next charge date from the current state.
//  2. Resolve the per-tenant reminder config (once per cycle, via activity) and
//     spawn a detached charge-reminder child per configured offset stage. Config
//     edits apply on the next cycle (a running cycle's reminders are fixed).
//  3. Wait for the charge time OR any of:
//     - signal subscription.paused / .resumed / .cancelled / .activated
//     - signal refresh-state
//     - signal cancel
//     If a signal fires, the wait result carries the latest Subscription
//     payload and the loop restarts with that state.
//  4. When the sleep wins, spawn the billing-cycle child workflow and await
//     the ChargeResult.
//  5. If the result is Pending, wait up to 1h for a webhook signal carrying
//     the resolved ChargeResult.
//  6. Hand the (possibly webhook-updated) ChargeResult to SubscriptionService.
//  7. Loop.
//
// Terminal exit: Cancelled, Expired, Completed, or a cancel signal.
//
// History rollover: when GetContinueAsNewSuggested() is true we restart with
// ContinueAsNew so the workflow history doesn't grow unbounded.
func SubscriptionWorkflow(ctx temporal.Context, input domain.Subscription) (domain.Subscription, error) {
	logger := temporal.GetLogger(ctx)
	sub := input
	cancelled := false

	logger.Info("SubscriptionWorkflow started", "subscriptionId", sub.Id, "orgId", sub.OrgId)

	if err := temporal.SetQueryHandler(ctx, "get-state", func() (domain.Subscription, error) {
		return sub, nil
	}); err != nil {
		return sub, err
	}

	pausedCh := temporal.GetSignalChannel(ctx, SignalSubscriptionPaused)
	resumedCh := temporal.GetSignalChannel(ctx, SignalSubscriptionResumed)
	cancelledCh := temporal.GetSignalChannel(ctx, SignalSubscriptionCancelled)
	activatedCh := temporal.GetSignalChannel(ctx, SignalSubscriptionActivated)
	refreshCh := temporal.GetSignalChannel(ctx, SignalRefreshState)
	runnerCancelCh := temporal.GetSignalChannel(ctx, SignalCancelRunner)
	webhookCh := temporal.GetSignalChannel(ctx, WebhookSignalName(sub.OrgId, sub.Id))

	// Drain control signals into the local subscription state via an
	// always-listening goroutine. Updates land synchronously so the wait
	// predicates below see them on the next reschedule.
	temporal.Go(ctx, func(gctx temporal.Context) {
		for {
			sel := temporal.NewSelector(gctx)
			var updated domain.Subscription
			sel.AddReceive(pausedCh, func(c temporal.ReceiveChannel, _ bool) {
				c.Receive(gctx, &updated)
				sub = updated
			})
			sel.AddReceive(resumedCh, func(c temporal.ReceiveChannel, _ bool) {
				c.Receive(gctx, &updated)
				sub = updated
			})
			sel.AddReceive(cancelledCh, func(c temporal.ReceiveChannel, _ bool) {
				c.Receive(gctx, &updated)
				sub = updated
			})
			sel.AddReceive(activatedCh, func(c temporal.ReceiveChannel, _ bool) {
				c.Receive(gctx, &updated)
				sub = updated
			})
			sel.AddReceive(refreshCh, func(c temporal.ReceiveChannel, _ bool) {
				c.Receive(gctx, &updated)
				sub = updated
			})
			sel.AddReceive(runnerCancelCh, func(c temporal.ReceiveChannel, _ bool) {
				c.Receive(gctx, &updated)
				cancelled = true
			})
			sel.Select(gctx)
		}
	})

	for {
		if cancelled || isTerminalSubscriptionStatus(sub.Status) {
			break
		}

		next := sub.GetNextChargeDate()
		if next.IsZero() {
			logger.Info("Subscription has no next charge date, ending workflow")
			break
		}

		// Activity-options context for this iteration's activity calls (the
		// reminder-config resolve below and the HandleChargeResult activity later).
		var act *activities.OrderActivities
		actCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
			StartToCloseTimeout: 5 * time.Minute,
			RetryPolicy: &temporalio.RetryPolicy{
				InitialInterval:    time.Minute,
				BackoffCoefficient: 1.2,
			},
		})

		// Reminders — resolve the per-tenant config ONCE per cycle (changes apply
		// next cycle), then schedule one detached child per offset stage.
		var reminderCfg domain.ReminderConfig
		if err := temporal.ExecuteActivity(actCtx, act.ResolveReminderConfig, sub.OrgId).Get(actCtx, &reminderCfg); err != nil {
			logger.Warn("ResolveReminderConfig failed; skipping reminders this cycle", "orgId", sub.OrgId, "err", err)
		}
		if reminderCfg.Enabled {
			for _, offset := range reminderCfg.Offsets {
				reminderAt := next.Add(-offset)
				if reminderAt.Before(temporal.Now(ctx)) {
					continue // this stage's lead time already passed for this cycle
				}
				reminderCtx := temporal.WithChildOptions(ctx, temporal.ChildWorkflowOptions{
					WorkflowID:            ReminderWorkflowID(sub.OrgId, sub.Id, sub.CyclesProcessed, offset),
					ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
					WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				})
				_ = temporal.ExecuteChildWorkflow(reminderCtx, SubscriptionChargeReminder, ReminderInput{
					Subscription: sub,
					ReminderAt:   reminderAt,
				}).GetChildWorkflowExecution().Get(ctx, nil)
			}
		}

		// Wait until the next charge time OR a control signal fires.
		wait := next.Sub(temporal.Now(ctx))
		if wait < time.Second {
			wait = time.Second
		}
		if _, err := temporal.AwaitWithTimeout(ctx, wait, func() bool {
			return cancelled ||
				sub.Status == domain.SubscriptionStatusPaused ||
				sub.Status == domain.SubscriptionStatusCancelled ||
				sub.Status == domain.SubscriptionStatusExpired ||
				temporal.GetInfo(ctx).GetContinueAsNewSuggested()
		}); err != nil {
			return sub, err
		}

		if cancelled {
			break
		}
		if temporal.GetInfo(ctx).GetContinueAsNewSuggested() {
			logger.Info("ContinueAsNewSuggested", "size", temporal.GetInfo(ctx).GetCurrentHistorySize())
			return sub, temporal.NewContinueAsNewError(ctx, SubscriptionWorkflow, sub)
		}
		if isTerminalSubscriptionStatus(sub.Status) {
			break
		}
		if sub.Status == domain.SubscriptionStatusPaused ||
			sub.Status == domain.SubscriptionStatusNonRenewing {
			continue
		}
		if !sub.IsRunning() {
			continue
		}
		// Charge date may have moved into the future (e.g. paused → activated
		// reset RenewsAt). Re-check before charging.
		if next.After(temporal.Now(ctx)) {
			continue
		}

		// Billing — child workflow, identical contract to Hatchet's DAG.
		billingCtx := temporal.WithChildOptions(ctx, temporal.ChildWorkflowOptions{
			WorkflowID:            BillingCycleWorkflowID(sub.OrgId, sub.Id, sub.CyclesProcessed),
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		})
		var chargeResult domain.ChargeResult
		if err := temporal.ExecuteChildWorkflow(billingCtx, BillingCycleWorkflow, BillingCycleInput{Subscription: sub}).
			Get(billingCtx, &chargeResult); err != nil {
			logger.Error("BillingCycleWorkflow failed", "err", err.Error())
			return sub, err
		}

		// On Pending, wait up to 1h for the per-(org, sub) webhook signal to
		// deliver the final ChargeResult.
		if chargeResult.Status == domain.PaymentStatusPending {
			selector := temporal.NewSelector(ctx)
			selector.AddReceive(webhookCh, func(c temporal.ReceiveChannel, _ bool) {
				var fromWebhook domain.ChargeResult
				c.Receive(ctx, &fromWebhook)
				chargeResult = fromWebhook
			})
			selector.AddFuture(temporal.NewTimer(ctx, time.Hour), func(temporal.Future) {
				logger.Warn("Timeout waiting for webhook signal", "subscriptionId", sub.Id)
			})
			selector.Select(ctx)
		}

		// Hand back to the service. Retry liberally — this only fails for DB
		// errors that should not crash the long-running workflow. Reuses the
		// iteration's actCtx/act declared above (the reminder-config resolve).
		var updated domain.Subscription
		if err := temporal.ExecuteActivity(actCtx, act.HandleChargeResult, sub, chargeResult).
			Get(actCtx, &updated); err != nil {
			logger.Error("HandleChargeResult failed", "err", err.Error())
			return sub, err
		}
		if updated.Id != "" {
			sub = updated
		}
	}

	logger.Info("SubscriptionWorkflow completed", "subscriptionId", sub.Id, "totalRevenue", sub.TotalRevenue)
	return sub, nil
}

func isTerminalSubscriptionStatus(s domain.SubscriptionStatus) bool {
	return s == domain.SubscriptionStatusCancelled ||
		s == domain.SubscriptionStatusExpired ||
		s == domain.SubscriptionStatusCompleted
}
