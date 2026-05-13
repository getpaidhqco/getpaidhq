package workflows

import (
	"time"

	"getpaidhq/internal/adapter/hatchet/steps"
	"getpaidhq/internal/core/domain"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewDunningRunnerWorkflow builds the per-campaign long-running dunning
// runner.
//
// Two-phase design:
//
//  1. **Immediate retries** — short, technical-failure-only retries with
//     intervals from config.ImmediateRetries.Intervals. Used when the initial
//     failure looks transient (rate limit, network, etc.).
//  2. **Progressive retries** — long, customer-driven waits with comms sent
//     before each attempt. Intervals from config.ProgressiveRetries.Intervals.
//
// Each attempt runs as a child "dunning-attempt" DAG; per-attempt outcome is
// fed back into DunningService.UpdateCampaignWithAttemptResult which owns the
// escalation policy (recover / suspend / cancel).
//
// Signals respected at every wait:
//   - dunning_signal:dunning.pause / .resume / .cancel
//   - dunning_pm_updated (triggers immediate retry)
//
// Terminal exits: campaign Status ∈ {recovered, failed, cancelled, expired}.
func NewDunningRunnerWorkflow(client *hatchet.Client, dunningSteps *steps.DunningSteps) *hatchet.StandaloneTask {
	return client.NewStandaloneDurableTask("dunning-runner",
		func(ctx hatchet.DurableContext, input DunningRunnerInput) (domain.DunningCampaign, error) {
			// Load the campaign's config snapshot (taken at campaign start) so
			// mid-flight config edits don't change cadence on a running
			// campaign. Falls back to the live config if no snapshot exists.
			config, err := dunningSteps.LoadConfigForCampaign(ctx, input.OrgId, input.CampaignId)
			if err != nil {
				return domain.DunningCampaign{}, err
			}

			pauseKey := DunningSignalKey("dunning.pause", input.OrgId, input.CampaignId)
			resumeKey := DunningSignalKey("dunning.resume", input.OrgId, input.CampaignId)
			cancelKey := DunningSignalKey("dunning.cancel", input.OrgId, input.CampaignId)
			pmUpdatedKey := DunningPaymentMethodUpdatedKey(input.OrgId, input.CampaignId)

			campaign := domain.DunningCampaign{
				OrgId: input.OrgId,
				Id:    input.CampaignId,
			}

			// Phase 1: immediate retries.
			if config.ImmediateRetries.Enabled && shouldUseImmediateRetries(input.InitialFailureReason, config.ImmediateRetries.FailureTypes) {
				for i := 0; i < config.ImmediateRetries.MaxAttempts && i < len(config.ImmediateRetries.Intervals); i++ {
					wait, err := domain.ParseDuration(config.ImmediateRetries.Intervals[i])
					if err != nil {
						wait = 5 * time.Minute
					}

					action, err := awaitDunningInterval(ctx, wait, pauseKey, resumeKey, cancelKey, pmUpdatedKey)
					if err != nil {
						return campaign, err
					}
					if action == dunningActionCancel {
						return campaign, nil
					}
					if action == dunningActionPaused {
						resumeAction, err := waitForResume(ctx, resumeKey, cancelKey)
						if err != nil {
							return campaign, err
						}
						if resumeAction == dunningActionCancel {
							return campaign, nil
						}
					}

					attempt, err := runDunningAttempt(ctx, client, input.OrgId, input.CampaignId, i+1, domain.DunningAttemptTypeImmediate)
					if err != nil {
						return campaign, err
					}

					updated, err := dunningSteps.UpdateCampaignWithAttemptResult(ctx, attempt, config, domain.DunningAttemptContext{
						AttemptNumber:            i + 1,
						WasSubscriptionSuspended: false,
					})
					if err != nil {
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

					action, err := awaitDunningInterval(ctx, wait, pauseKey, resumeKey, cancelKey, pmUpdatedKey)
					if err != nil {
						return campaign, err
					}
					if action == dunningActionCancel {
						return campaign, nil
					}
					if action == dunningActionPaused {
						resumeAction, err := waitForResume(ctx, resumeKey, cancelKey)
						if err != nil {
							return campaign, err
						}
						if resumeAction == dunningActionCancel {
							return campaign, nil
						}
					}

					attemptNumber := i + 1
					_ = dunningSteps.SendCommunication(ctx, input.OrgId, input.CampaignId, attemptNumber)

					attempt, err := runDunningAttempt(ctx, client, input.OrgId, input.CampaignId, attemptNumber, domain.DunningAttemptTypeProgressive)
					if err != nil {
						return campaign, err
					}

					wasSuspended := config.EscalationRules.SuspendAfterAttempt > 0 && attemptNumber >= config.EscalationRules.SuspendAfterAttempt
					updated, err := dunningSteps.UpdateCampaignWithAttemptResult(ctx, attempt, config, domain.DunningAttemptContext{
						AttemptNumber:            attemptNumber,
						WasSubscriptionSuspended: wasSuspended,
					})
					if err != nil {
						return campaign, err
					}
					campaign = updated
					if isDunningTerminal(campaign.Status) {
						return campaign, nil
					}
				}
			}

			// Exhausted all attempts. Cancel the subscription too — without
			// this, configs where MaxAttempts < CancelAfterAttempt (or
			// CancelAfterAttempt == 0) silently leave Active subscriptions
			// behind that no future billing-cycle can ever charge.
			final, err := dunningSteps.FailCampaignAndCancelSubscription(ctx, input.OrgId, input.CampaignId, "all_attempts_failed")
			if err != nil {
				return campaign, err
			}
			return final, nil
		},
	)
}

// dunningAction describes what woke the durable wait up.
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
func awaitDunningInterval(ctx hatchet.DurableContext, wait time.Duration, pauseKey, resumeKey, cancelKey, pmUpdatedKey string) (dunningAction, error) {
	if wait < time.Second {
		wait = time.Second
	}
	res, err := ctx.WaitFor(hatchet.OrCondition(
		hatchet.SleepCondition(wait),
		hatchet.UserEventCondition(pauseKey, ""),
		hatchet.UserEventCondition(resumeKey, ""),
		hatchet.UserEventCondition(cancelKey, ""),
		hatchet.UserEventCondition(pmUpdatedKey, ""),
	))
	if err != nil {
		return dunningActionTimer, err
	}
	keys := res.Keys()
	if containsKey(keys, cancelKey) {
		return dunningActionCancel, nil
	}
	if containsKey(keys, pauseKey) {
		return dunningActionPaused, nil
	}
	if containsKey(keys, pmUpdatedKey) {
		return dunningActionImmediateRetry, nil
	}
	return dunningActionTimer, nil
}

// waitForResume blocks until a resume or cancel signal fires. Returns the
// action that woke it so the caller can stop the runner on cancel rather than
// proceed with the next attempt.
func waitForResume(ctx hatchet.DurableContext, resumeKey, cancelKey string) (dunningAction, error) {
	for {
		res, err := ctx.WaitFor(hatchet.OrCondition(
			hatchet.UserEventCondition(resumeKey, ""),
			hatchet.UserEventCondition(cancelKey, ""),
		))
		if err != nil {
			return dunningActionTimer, err
		}
		keys := res.Keys()
		if containsKey(keys, cancelKey) {
			return dunningActionCancel, nil
		}
		if containsKey(keys, resumeKey) {
			return dunningActionTimer, nil
		}
	}
}

func runDunningAttempt(ctx hatchet.DurableContext, client *hatchet.Client, orgId, campaignId string, attemptNumber int, attemptType domain.DunningAttemptType) (domain.DunningAttempt, error) {
	res, err := client.Run(ctx, "dunning-attempt", DunningAttemptInput{
		OrgId:         orgId,
		CampaignId:    campaignId,
		AttemptNumber: attemptNumber,
		AttemptType:   attemptType,
	}, hatchet.WithRunKey(DunningAttemptRunKey(orgId, campaignId, attemptNumber)))
	if err != nil {
		return domain.DunningAttempt{}, err
	}
	var attempt domain.DunningAttempt
	if err := res.TaskOutput("execute-attempt").Into(&attempt); err != nil {
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
