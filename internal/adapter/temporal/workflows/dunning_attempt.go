package workflows

import (
	"time"

	temporalio "go.temporal.io/sdk/temporal"
	temporal "go.temporal.io/sdk/workflow"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
)

// DunningAttemptWorkflow runs a single dunning retry. Mirrors
// internal/adapter/hatchet/workflows/dunning_attempt.go.
//
// The runner spawns one of these for each retry; the attempt result is read
// back by the runner and handed to DunningService.UpdateCampaignWithAttemptResult
// which owns the escalation policy.
func DunningAttemptWorkflow(ctx temporal.Context, input DunningAttemptInput) (domain.DunningAttempt, error) {
	var act *activities.DunningActivities

	actCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 1.5,
			MaximumAttempts:    10,
			MaximumInterval:    5 * time.Minute,
		},
	})

	var attempt domain.DunningAttempt
	if err := temporal.ExecuteActivity(actCtx, act.ExecuteAttempt, input.OrgId, input.CampaignId, input.AttemptType).
		Get(actCtx, &attempt); err != nil {
		return domain.DunningAttempt{}, err
	}
	return attempt, nil
}
