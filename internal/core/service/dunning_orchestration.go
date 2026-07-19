package service

import (
	"context"
	"encoding/json"
	"getpaidhq/internal/lib/errors"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// DunningOrchestrationService wraps the narrow DunningService and adds
// workflow-engine signaling. HTTP handlers depend on this service; Hatchet
// steps depend on the narrow service via port.DunningService.
//
// On construction it also subscribes to the
// subscription.payment.charge.failed topic so that whenever a charge fails
// (anywhere in the system) a dunning campaign is started automatically.
type DunningOrchestrationService struct {
	*DunningService
	dunningEngine port.DunningEngine
	pubsub        port.PubSub
	errorReporter errors.ErrorReporter
	logger        port.Logger
}

func NewDunningOrchestrationService(
	dunning *DunningService,
	dunningEngine port.DunningEngine,
	pubsub port.PubSub,
	errorReporter errors.ErrorReporter,
	logger port.Logger,
) (*DunningOrchestrationService, error) {
	svc := &DunningOrchestrationService{
		DunningService: dunning,
		dunningEngine:  dunningEngine,
		pubsub:         pubsub,
		errorReporter:  errorReporter,
		logger:         logger,
	}

	logger.Debugf("[DunningOrchestrationService] Subscribing to %s", port.TopicSubscriptionPaymentChargeFailed)
	if _, err := pubsub.Subscribe(port.TopicSubscriptionPaymentChargeFailed, svc.HandleSubscriptionChargeFailure); err != nil {
		return nil, err
	}

	return svc, nil
}

// HandleSubscriptionChargeFailure starts a dunning workflow when a subscription
// charge fails.
func (s *DunningOrchestrationService) HandleSubscriptionChargeFailure(topic string, data []byte) {
	s.logger.Infof("[DunningOrchestrationService] received %s", topic)

	var envelope port.PubSubPayload
	if err := json.Unmarshal(data, &envelope); err != nil {
		s.logger.Errorf("Failed to unmarshal envelope: %v", err)
		return
	}

	// The publisher (SubscriptionService.HandleSubscriptionChargeFailure)
	// publishes a map containing "subscription" and "charge_result".
	payloadBytes, err := json.Marshal(envelope.Data)
	if err != nil {
		s.logger.Errorf("Failed to marshal payload data: %v", err)
		return
	}
	var payload struct {
		Subscription domain.Subscription `json:"subscription"`
		ChargeResult domain.ChargeResult `json:"charge_result"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		s.logger.Errorf("Failed to unmarshal charge-failure payload: %v", err)
		return
	}

	if _, err := s.StartDunningWorkflow(context.Background(), port.StartDunningWorkflowInput{
		OrgId:                payload.Subscription.OrgId,
		SubscriptionId:       payload.Subscription.Id,
		CustomerId:           payload.Subscription.CustomerId,
		FailedAmount:         payload.ChargeResult.Amount,
		Currency:             payload.ChargeResult.Currency,
		InitialFailureReason: payload.ChargeResult.ErrorReason,
		PaymentResult:        payload.ChargeResult,
		Metadata: map[string]string{
			"triggered_by":    "subscription_charge_failure",
			"failure_code":    payload.ChargeResult.ErrorCode,
			"subscription_id": payload.Subscription.Id,
		},
	}); err != nil {
		s.logger.Errorf("Failed to start dunning workflow: %v", err)
		s.errorReporter.ReportError(context.Background(), err, map[string]any{
			"operation":       "start_dunning_workflow",
			"org_id":          payload.Subscription.OrgId,
			"subscription_id": payload.Subscription.Id,
		})
	}
}

// StartDunningWorkflow creates a campaign and asks the engine to start a
// dunning workflow run for it. The campaign's WorkflowId / WorkflowRunId are
// updated with the engine-provided handles so the orchestrator can address
// the run later.
//
// The dunning config is resolved here (not inside the runner) so the snapshot
// stored on the campaign reflects the policy in force at campaign start; the
// runner reads back the snapshot instead of re-resolving live config on every
// resume.
func (s *DunningOrchestrationService) StartDunningWorkflow(ctx context.Context, input port.StartDunningWorkflowInput) (domain.DunningCampaign, error) {
	s.logger.Info("Starting dunning workflow", "orgId", input.OrgId, "subscriptionId", input.SubscriptionId)

	resolved, err := s.ResolveConfig(ctx, input.OrgId)
	if err != nil {
		s.logger.Error("Failed to resolve dunning config at campaign start", "err", err.Error())
		resolved = domain.DefaultDunningConfig()
	}
	snapshot, err := configToMap(resolved)
	if err != nil {
		s.logger.Error("Failed to marshal dunning config snapshot", "err", err.Error())
		snapshot = nil
	}

	campaign, err := s.CreateCampaign(ctx, port.CreateDunningCampaignInput{
		OrgId:                input.OrgId,
		SubscriptionId:       input.SubscriptionId,
		CustomerId:           input.CustomerId,
		FailedAmount:         input.FailedAmount,
		Currency:             input.Currency,
		InitialFailureReason: input.InitialFailureReason,
		ParentWorkflowId:     input.ParentWorkflowId,
		ConfigSnapshot:       snapshot,
		Metadata:             input.Metadata,
	})
	if err != nil {
		return domain.DunningCampaign{}, err
	}

	workflowId, runId, err := s.dunningEngine.StartDunningWorkflow(ctx, input)
	if err != nil {
		s.logger.Error("Failed to start dunning engine workflow", "err", err.Error())
		return campaign, errors.NewCustomError(errors.InternalError, "Failed to start dunning workflow", err)
	}

	campaign.WorkflowId = workflowId
	campaign.WorkflowRunId = runId
	updated, err := s.UpdateCampaign(ctx, campaign)
	if err != nil {
		s.logger.Error("Failed to store workflow handle on campaign", "err", err.Error())
		return campaign, nil
	}
	return updated, nil
}

// PauseCampaign / ResumeCampaign / CancelCampaign override the narrow service
// methods to also signal the engine.
func (s *DunningOrchestrationService) PauseCampaign(ctx context.Context, input port.PauseDunningCampaignInput) (domain.DunningCampaign, error) {
	c, err := s.DunningService.PauseCampaign(ctx, input)
	if err != nil {
		return c, err
	}
	if err := s.dunningEngine.SignalDunningWorkflow(ctx, "dunning.pause", c, input.Reason); err != nil {
		s.logger.Error("Failed to signal dunning.pause", "err", err.Error())
	}
	return c, nil
}

func (s *DunningOrchestrationService) ResumeCampaign(ctx context.Context, input port.ResumeDunningCampaignInput) (domain.DunningCampaign, error) {
	c, err := s.DunningService.ResumeCampaign(ctx, input)
	if err != nil {
		return c, err
	}
	if err := s.dunningEngine.SignalDunningWorkflow(ctx, "dunning.resume", c, input.Reason); err != nil {
		s.logger.Error("Failed to signal dunning.resume", "err", err.Error())
	}
	return c, nil
}

func (s *DunningOrchestrationService) CancelCampaign(ctx context.Context, input port.CancelDunningCampaignInput) (domain.DunningCampaign, error) {
	c, err := s.DunningService.CancelCampaign(ctx, input)
	if err != nil {
		return c, err
	}
	if err := s.dunningEngine.CancelDunningWorkflow(ctx, c); err != nil {
		s.logger.Error("Failed to cancel dunning workflow", "err", err.Error())
	}
	return c, nil
}
