package workflows

import (
	"time"

	"getpaidhq/internal/adapter/hatchet/steps"
	"getpaidhq/internal/core/domain"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewDunningAttemptWorkflow builds the per-attempt DAG. The dunning runner
// spawns one of these for each retry: it executes the charge and lets the
// service layer apply the escalation policy.
//
// The runner reads the attempt result back via TaskOutput("execute-attempt").
func NewDunningAttemptWorkflow(client *hatchet.Client, dunningSteps *steps.DunningSteps) *hatchet.Workflow {
	wf := client.NewWorkflow("dunning-attempt")

	wf.NewTask("execute-attempt",
		func(ctx hatchet.Context, input DunningAttemptInput) (domain.DunningAttempt, error) {
			return dunningSteps.ExecuteAttempt(ctx, input.OrgId, input.CampaignId, input.AttemptType)
		},
		hatchet.WithExecutionTimeout(60*time.Second),
		hatchet.WithRetries(10),
		hatchet.WithRetryBackoff(1.5, 300),
	)

	return wf
}
