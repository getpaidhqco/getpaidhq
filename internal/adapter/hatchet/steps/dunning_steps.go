package steps

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// DunningSteps is the Hatchet-side glue for the dunning workflow. Each method
// is a thin wrapper around port.DunningService so the workflow body holds no
// business rules.
type DunningSteps struct {
	logger         port.Logger
	dunningService port.DunningService
}

func NewDunningSteps(logger port.Logger, dunningService port.DunningService) *DunningSteps {
	return &DunningSteps{
		logger:         logger,
		dunningService: dunningService,
	}
}

// LoadConfigForCampaign reads the config snapshot stored on the campaign at
// start time; falls back to the org's live config if none was snapshotted
// (e.g. for campaigns started before snapshotting existed).
func (s *DunningSteps) LoadConfigForCampaign(ctx context.Context, orgId, campaignId string) (domain.DunningConfig, error) {
	s.logger.Info("LoadDunningConfigForCampaign", "orgId", orgId, "campaignId", campaignId)
	return s.dunningService.LoadConfigForCampaign(ctx, orgId, campaignId)
}

func (s *DunningSteps) ExecuteAttempt(ctx context.Context, orgId, campaignId string, attemptType domain.DunningAttemptType) (domain.DunningAttempt, error) {
	s.logger.Info("ExecuteDunningAttempt", "orgId", orgId, "campaignId", campaignId, "type", string(attemptType))
	return s.dunningService.ExecuteAttempt(ctx, orgId, campaignId, attemptType)
}

func (s *DunningSteps) UpdateCampaignWithAttemptResult(ctx context.Context, attempt domain.DunningAttempt, config domain.DunningConfig, attemptContext domain.DunningAttemptContext) (domain.DunningCampaign, error) {
	s.logger.Info("UpdateCampaignWithAttemptResult", "campaignId", attempt.DunningCampaignId, "attemptNumber", attempt.AttemptNumber)
	return s.dunningService.UpdateCampaignWithAttemptResult(ctx, attempt, config, attemptContext)
}

func (s *DunningSteps) SendCommunication(ctx context.Context, orgId, campaignId string, attemptNumber int) error {
	s.logger.Info("SendDunningCommunication", "campaignId", campaignId, "attempt", attemptNumber)
	return s.dunningService.SendCommunication(ctx, orgId, campaignId, attemptNumber)
}

func (s *DunningSteps) MarkCampaignFailed(ctx context.Context, orgId, campaignId, reason string) (domain.DunningCampaign, error) {
	s.logger.Info("MarkCampaignFailed", "campaignId", campaignId, "reason", reason)
	return s.dunningService.MarkCampaignFailed(ctx, orgId, campaignId, reason)
}

// FailCampaignAndCancelSubscription is the terminal exit when retries exhaust
// without an explicit cancellation threshold catching it first — both
// mutations happen together so the subscription doesn't outlive its dunning.
func (s *DunningSteps) FailCampaignAndCancelSubscription(ctx context.Context, orgId, campaignId, reason string) (domain.DunningCampaign, error) {
	s.logger.Info("FailCampaignAndCancelSubscription", "campaignId", campaignId, "reason", reason)
	return s.dunningService.FailCampaignAndCancelSubscription(ctx, orgId, campaignId, reason)
}
