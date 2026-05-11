package steps

import (
	"context"

	"payloop/internal/core/domain"
	"payloop/internal/core/port"
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

func (s *DunningSteps) ResolveConfig(ctx context.Context, orgId string) (domain.DunningConfig, error) {
	s.logger.Info("ResolveDunningConfig", "orgId", orgId)
	return s.dunningService.ResolveConfig(ctx, orgId)
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

func (s *DunningSteps) PauseCampaign(ctx context.Context, orgId, campaignId string) (domain.DunningCampaign, error) {
	s.logger.Info("PauseDunningCampaign", "campaignId", campaignId)
	return s.dunningService.PauseCampaign(ctx, domain.PauseDunningCampaignInput{
		OrgId:      orgId,
		CampaignId: campaignId,
		Reason:     "engine_pause",
	})
}

func (s *DunningSteps) ResumeCampaign(ctx context.Context, orgId, campaignId string) (domain.DunningCampaign, error) {
	s.logger.Info("ResumeDunningCampaign", "campaignId", campaignId)
	return s.dunningService.ResumeCampaign(ctx, domain.ResumeDunningCampaignInput{
		OrgId:      orgId,
		CampaignId: campaignId,
		Reason:     "engine_resume",
	})
}

func (s *DunningSteps) CancelCampaign(ctx context.Context, orgId, campaignId string) (domain.DunningCampaign, error) {
	s.logger.Info("CancelDunningCampaign", "campaignId", campaignId)
	return s.dunningService.CancelCampaign(ctx, domain.CancelDunningCampaignInput{
		OrgId:      orgId,
		CampaignId: campaignId,
		Reason:     "engine_cancel",
	})
}

func (s *DunningSteps) TriggerImmediateRetry(ctx context.Context, orgId, campaignId string) (domain.DunningAttempt, error) {
	s.logger.Info("TriggerImmediateRetry", "campaignId", campaignId)
	return s.dunningService.TriggerManualAttempt(ctx, domain.TriggerManualAttemptInput{
		OrgId:       orgId,
		CampaignId:  campaignId,
		TriggeredBy: "payment_method_updated",
	})
}

func (s *DunningSteps) MarkCampaignFailed(ctx context.Context, orgId, campaignId, reason string) (domain.DunningCampaign, error) {
	s.logger.Info("MarkCampaignFailed", "campaignId", campaignId, "reason", reason)
	return s.dunningService.MarkCampaignFailed(ctx, orgId, campaignId, reason)
}
