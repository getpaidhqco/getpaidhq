package workflows

import (
	"time"

	"go.temporal.io/api/enums/v1"
	temporalio "go.temporal.io/sdk/temporal"
	temporal "go.temporal.io/sdk/workflow"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
)

// DunningRunnerWorkflow is the per-campaign long-running dunning runner.
// Mirrors internal/adapter/hatchet/workflows/dunning_runner.go.
//
// Two-phase design:
//
//  1. **Immediate retries** — short, technical-failure-only retries with
//     intervals from config.ImmediateRetries.Intervals. Used when the initial
//     failure looks transient (rate limit, network, etc.).
//  2. **Progressive retries** — long, customer-driven waits with comms sent
//     before each attempt. Intervals from config.ProgressiveRetries.Intervals.
//
// Each attempt runs as a child workflow; per-attempt outcome is fed back into
// DunningService.UpdateCampaignWithAttemptResult which owns the escalation
// policy.
//
// Signals respected at every wait:
//   - dunning.pause / dunning.resume / dunning.cancel
//   - dunning.payment_method_updated (triggers immediate retry)
//
// Terminal exits: campaign Status ∈ {recovered, failed, cancelled, expired}.
func DunningRunnerWorkflow(ctx temporal.Context, input DunningRunnerInput) (domain.DunningCampaign, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("DunningRunnerWorkflow started", "orgId", input.OrgId, "campaignId", input.CampaignId)

	var act *activities.DunningActivities

	loadCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 1.5,
			MaximumAttempts:    5,
		},
	})
	var config domain.DunningConfig
	if err := temporal.ExecuteActivity(loadCtx, act.LoadConfigForCampaign, input.OrgId, input.CampaignId).
		Get(loadCtx, &config); err != nil {
		return domain.DunningCampaign{}, err
	}

	pauseCh := temporal.GetSignalChannel(ctx, SignalDunningPause)
	resumeCh := temporal.GetSignalChannel(ctx, SignalDunningResume)
	cancelCh := temporal.GetSignalChannel(ctx, SignalDunningCancel)
	pmUpdatedCh := temporal.GetSignalChannel(ctx, SignalDunningPaymentMethodUpd)

	campaign := domain.DunningCampaign{OrgId: input.OrgId, Id: input.CampaignId}

	// Phase 1: immediate retries.
	if config.ImmediateRetries.Enabled && shouldUseImmediateRetries(input.InitialFailureReason, config.ImmediateRetries.FailureTypes) {
		for i := 0; i < config.ImmediateRetries.MaxAttempts && i < len(config.ImmediateRetries.Intervals); i++ {
			wait, err := domain.ParseDuration(config.ImmediateRetries.Intervals[i])
			if err != nil {
				wait = 5 * time.Minute
			}

			action := awaitDunningInterval(ctx, wait, pauseCh, resumeCh, cancelCh, pmUpdatedCh)
			if action == dunningActionCancel {
				return campaign, nil
			}
			if action == dunningActionPaused {
				if waitForResume(ctx, resumeCh, cancelCh) == dunningActionCancel {
					return campaign, nil
				}
			}

			attempt, err := runDunningAttempt(ctx, input.OrgId, input.CampaignId, i+1, domain.DunningAttemptTypeImmediate)
			if err != nil {
				return campaign, err
			}

			updateCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
				StartToCloseTimeout: 30 * time.Second,
				RetryPolicy: &temporalio.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 1.5,
					MaximumAttempts:    5,
				},
			})
			var updated domain.DunningCampaign
			if err := temporal.ExecuteActivity(updateCtx, act.UpdateCampaignWithAttemptResult, attempt, config, domain.DunningAttemptContext{
				AttemptNumber:            i + 1,
				WasSubscriptionSuspended: false,
			}).Get(updateCtx, &updated); err != nil {
				return campaign, err
			}
			campaign = updated
			if isDunningTerminal(campaign.Status) {
				return campaign, nil
			}
		}
	}

	// Phase 2: progressive retries.
	if config.ProgressiveRetries.Enabled {
		for i := 0; i < config.ProgressiveRetries.MaxAttempts && i < len(config.ProgressiveRetries.Intervals); i++ {
			wait, err := domain.ParseDuration(config.ProgressiveRetries.Intervals[i])
			if err != nil {
				wait = 3 * 24 * time.Hour
			}

			action := awaitDunningInterval(ctx, wait, pauseCh, resumeCh, cancelCh, pmUpdatedCh)
			if action == dunningActionCancel {
				return campaign, nil
			}
			if action == dunningActionPaused {
				if waitForResume(ctx, resumeCh, cancelCh) == dunningActionCancel {
					return campaign, nil
				}
			}

			attemptNumber := i + 1

			commCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
				StartToCloseTimeout: 30 * time.Second,
				RetryPolicy: &temporalio.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 1.5,
					MaximumAttempts:    3,
				},
			})
			_ = temporal.ExecuteActivity(commCtx, act.SendCommunication, input.OrgId, input.CampaignId, attemptNumber).
				Get(commCtx, nil)

			attempt, err := runDunningAttempt(ctx, input.OrgId, input.CampaignId, attemptNumber, domain.DunningAttemptTypeProgressive)
			if err != nil {
				return campaign, err
			}

			wasSuspended := config.EscalationRules.SuspendAfterAttempt > 0 && attemptNumber >= config.EscalationRules.SuspendAfterAttempt
			updateCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
				StartToCloseTimeout: 30 * time.Second,
				RetryPolicy: &temporalio.RetryPolicy{
					InitialInterval:    time.Second,
					BackoffCoefficient: 1.5,
					MaximumAttempts:    5,
				},
			})
			var updated domain.DunningCampaign
			if err := temporal.ExecuteActivity(updateCtx, act.UpdateCampaignWithAttemptResult, attempt, config, domain.DunningAttemptContext{
				AttemptNumber:            attemptNumber,
				WasSubscriptionSuspended: wasSuspended,
			}).Get(updateCtx, &updated); err != nil {
				return campaign, err
			}
			campaign = updated
			if isDunningTerminal(campaign.Status) {
				return campaign, nil
			}
		}
	}

	// Exhausted all attempts — cancel the subscription too. See
	// internal/adapter/hatchet/workflows/dunning_runner.go for the same
	// terminal-exit reasoning.
	finalCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 1.5,
			MaximumAttempts:    5,
		},
	})
	var final domain.DunningCampaign
	if err := temporal.ExecuteActivity(finalCtx, act.FailCampaignAndCancelSubscription, input.OrgId, input.CampaignId, "all_attempts_failed").
		Get(finalCtx, &final); err != nil {
		return campaign, err
	}
	return final, nil
}

type dunningAction int

const (
	dunningActionTimer dunningAction = iota
	dunningActionPaused
	dunningActionCancel
	dunningActionImmediateRetry
)

// awaitDunningInterval sleeps until the next attempt is due OR a control
// signal fires. Pause/resume is signalled here too; the caller is expected to
// loop on dunningActionPaused via waitForResume to model the pause/resume
// state.
func awaitDunningInterval(ctx temporal.Context, wait time.Duration, pauseCh, resumeCh, cancelCh, pmUpdatedCh temporal.ReceiveChannel) dunningAction {
	if wait < time.Second {
		wait = time.Second
	}

	result := dunningActionTimer
	timer := temporal.NewTimer(ctx, wait)
	selector := temporal.NewSelector(ctx)
	selector.AddFuture(timer, func(temporal.Future) { result = dunningActionTimer })
	selector.AddReceive(pauseCh, func(c temporal.ReceiveChannel, _ bool) {
		var v any
		c.Receive(ctx, &v)
		result = dunningActionPaused
	})
	selector.AddReceive(resumeCh, func(c temporal.ReceiveChannel, _ bool) {
		var v any
		c.Receive(ctx, &v)
		// Stray resume during a wait — treat as a no-op; the timer will still
		// resolve eventually.
		result = dunningActionTimer
	})
	selector.AddReceive(cancelCh, func(c temporal.ReceiveChannel, _ bool) {
		var v any
		c.Receive(ctx, &v)
		result = dunningActionCancel
	})
	selector.AddReceive(pmUpdatedCh, func(c temporal.ReceiveChannel, _ bool) {
		var v any
		c.Receive(ctx, &v)
		result = dunningActionImmediateRetry
	})
	selector.Select(ctx)
	return result
}

// waitForResume blocks until a resume or cancel signal fires. Returns the
// action that woke it so the caller can stop the runner on cancel rather than
// proceed with the next attempt.
func waitForResume(ctx temporal.Context, resumeCh, cancelCh temporal.ReceiveChannel) dunningAction {
	for {
		result := dunningActionTimer
		selector := temporal.NewSelector(ctx)
		selector.AddReceive(resumeCh, func(c temporal.ReceiveChannel, _ bool) {
			var v any
			c.Receive(ctx, &v)
			result = dunningActionTimer
		})
		selector.AddReceive(cancelCh, func(c temporal.ReceiveChannel, _ bool) {
			var v any
			c.Receive(ctx, &v)
			result = dunningActionCancel
		})
		selector.Select(ctx)
		if result == dunningActionCancel || result == dunningActionTimer {
			return result
		}
	}
}

func runDunningAttempt(ctx temporal.Context, orgId, campaignId string, attemptNumber int, attemptType domain.DunningAttemptType) (domain.DunningAttempt, error) {
	childCtx := temporal.WithChildOptions(ctx, temporal.ChildWorkflowOptions{
		WorkflowID:            DunningAttemptWorkflowID(orgId, campaignId, attemptNumber),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	})
	var attempt domain.DunningAttempt
	if err := temporal.ExecuteChildWorkflow(childCtx, DunningAttemptWorkflow, DunningAttemptInput{
		OrgId:         orgId,
		CampaignId:    campaignId,
		AttemptNumber: attemptNumber,
		AttemptType:   attemptType,
	}).Get(childCtx, &attempt); err != nil {
		return domain.DunningAttempt{}, err
	}
	return attempt, nil
}

func isDunningTerminal(s domain.DunningStatus) bool {
	return s == domain.DunningStatusRecovered ||
		s == domain.DunningStatusFailed ||
		s == domain.DunningStatusCancelled ||
		s == domain.DunningStatusExpired
}

func shouldUseImmediateRetries(initialFailureReason string, immediateFailureTypes []string) bool {
	if initialFailureReason == "" {
		return false
	}
	for _, t := range immediateFailureTypes {
		if t == initialFailureReason {
			return true
		}
	}
	return false
}
