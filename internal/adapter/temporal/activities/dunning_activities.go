package activities

import (
	"context"

	"go.temporal.io/sdk/activity"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// DunningActivities is the Temporal mirror of
// internal/adapter/hatchet/steps/dunning_steps.go. Each method is a thin
// wrapper around port.DunningService so the workflow body holds no business
// rules.
type DunningActivities struct {
	dunningService port.DunningService
}

func NewDunningActivities(dunningService port.DunningService) DunningActivities {
	return DunningActivities{dunningService: dunningService}
}

// LoadConfigForCampaign reads the config snapshot stored on the campaign at
// start time; falls back to the org's live config if none was snapshotted.
func (a *DunningActivities) LoadConfigForCampaign(ctx context.Context, orgId, campaignId string) (domain.DunningConfig, error) {
	activity.GetLogger(ctx).Info("LoadConfigForCampaign", "orgId", orgId, "campaignId", campaignId)
	return a.dunningService.LoadConfigForCampaign(ctx, orgId, campaignId)
}

func (a *DunningActivities) ExecuteAttempt(ctx context.Context, orgId, campaignId string, attemptType domain.DunningAttemptType) (domain.DunningAttempt, error) {
	activity.GetLogger(ctx).Info("ExecuteAttempt", "orgId", orgId, "campaignId", campaignId, "type", string(attemptType))
	return a.dunningService.ExecuteAttempt(ctx, orgId, campaignId, attemptType)
}

func (a *DunningActivities) UpdateCampaignWithAttemptResult(ctx context.Context, attempt domain.DunningAttempt, config domain.DunningConfig, attemptContext domain.DunningAttemptContext) (domain.DunningCampaign, error) {
	activity.GetLogger(ctx).Info("UpdateCampaignWithAttemptResult", "campaignId", attempt.DunningCampaignId, "attemptNumber", attempt.AttemptNumber)
	return a.dunningService.UpdateCampaignWithAttemptResult(ctx, attempt, config, attemptContext)
}

func (a *DunningActivities) SendCommunication(ctx context.Context, orgId, campaignId string, attemptNumber int) error {
	activity.GetLogger(ctx).Info("SendCommunication", "campaignId", campaignId, "attempt", attemptNumber)
	return a.dunningService.SendCommunication(ctx, orgId, campaignId, attemptNumber)
}

func (a *DunningActivities) MarkCampaignFailed(ctx context.Context, orgId, campaignId, reason string) (domain.DunningCampaign, error) {
	activity.GetLogger(ctx).Info("MarkCampaignFailed", "campaignId", campaignId, "reason", reason)
	return a.dunningService.MarkCampaignFailed(ctx, orgId, campaignId, reason)
}

func (a *DunningActivities) FailCampaignAndCancelSubscription(ctx context.Context, orgId, campaignId, reason string) (domain.DunningCampaign, error) {
	activity.GetLogger(ctx).Info("FailCampaignAndCancelSubscription", "campaignId", campaignId, "reason", reason)
	return a.dunningService.FailCampaignAndCancelSubscription(ctx, orgId, campaignId, reason)
}
