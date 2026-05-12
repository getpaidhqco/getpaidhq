package temporal

import (
	"context"
	"errors"

	"payloop/internal/core/domain"
)

// errDunningNotImplemented is returned by the Temporal adapter for every
// dunning-engine call. Dunning is currently a Hatchet-only feature; running
// payloop on Temporal disables the dunning workflow path. The orchestration
// service still creates campaigns in the DB but no workflow is started.
var errDunningNotImplemented = errors.New("dunning workflows are not implemented on the Temporal engine; switch WORKFLOW_ENGINE=hatchet")

func (t Temporal) StartDunningWorkflow(ctx context.Context, input domain.StartDunningWorkflowInput) (string, string, error) {
	t.logger.Warnf("StartDunningWorkflow called on Temporal adapter — dunning is Hatchet-only")
	return "", "", errDunningNotImplemented
}

func (t Temporal) SignalDunningWorkflow(ctx context.Context, signal string, campaign domain.DunningCampaign, payload any) error {
	t.logger.Warnf("SignalDunningWorkflow called on Temporal adapter — dunning is Hatchet-only")
	return errDunningNotImplemented
}

func (t Temporal) CancelDunningWorkflow(ctx context.Context, campaign domain.DunningCampaign) error {
	t.logger.Warnf("CancelDunningWorkflow called on Temporal adapter — dunning is Hatchet-only")
	return errDunningNotImplemented
}
