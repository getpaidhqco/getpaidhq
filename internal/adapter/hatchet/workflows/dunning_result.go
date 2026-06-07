package workflows

import (
	"time"

	"getpaidhq/internal/adapter/hatchet/steps"
	"getpaidhq/internal/core/domain"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewDunningResultWorkflow builds the per-attempt result-application child. The
// dunning runner spawns one (run-key deduped) after each attempt to apply the
// escalation policy (recover / suspend / cancel) and publish the resulting
// domain events.
//
// Wrapping this in a child run means Hatchet memoizes it: the status
// transitions and — more importantly — the downstream event publishes fire
// exactly once even when the runner is evicted and replayed mid-campaign.
func NewDunningResultWorkflow(client *hatchet.Client, dunningSteps *steps.DunningSteps) *hatchet.Workflow {
	wf := client.NewWorkflow("dunning-result")

	wf.NewTask("apply-result",
		func(ctx hatchet.Context, input DunningResultInput) (domain.DunningCampaign, error) {
			return dunningSteps.UpdateCampaignWithAttemptResult(ctx, input.Attempt, input.Config, input.AttemptContext)
		},
		hatchet.WithExecutionTimeout(30*time.Second),
		hatchet.WithRetries(5),
		hatchet.WithRetryBackoff(1.5, 60),
	)

	return wf
}
