package workflows

import (
	"time"

	"getpaidhq/internal/adapter/hatchet/steps"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewDunningCommunicationWorkflow builds the per-attempt communication child.
// The dunning runner spawns one (run-key deduped) before each progressive
// attempt. Wrapping the side-effect in a child run means Hatchet memoizes it,
// so the customer comm fires exactly once even when the runner is evicted and
// replayed mid-campaign.
func NewDunningCommunicationWorkflow(client *hatchet.Client, dunningSteps *steps.DunningSteps) *hatchet.Workflow {
	wf := client.NewWorkflow("dunning-communication")

	wf.NewTask("send-communication",
		func(ctx hatchet.Context, input DunningCommunicationInput) (DunningCommunicationOutput, error) {
			if err := dunningSteps.SendCommunication(ctx, input.OrgId, input.CampaignId, input.AttemptNumber); err != nil {
				return DunningCommunicationOutput{}, err
			}
			return DunningCommunicationOutput{Sent: true}, nil
		},
		hatchet.WithExecutionTimeout(30*time.Second),
		hatchet.WithRetries(3),
		hatchet.WithRetryBackoff(1.5, 60),
	)

	return wf
}
