package workflows

import (
	"getpaidhq/internal/adapter/hatchet/steps"
	"getpaidhq/internal/core/port"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// NewOutgoingWebhookWorkflow delivers a single outbound webhook with retry
// (MaximumAttempts: 5, InitialInterval: 1 minute, BackoffCoefficient: 1.0).
func NewOutgoingWebhookWorkflow(client *hatchet.Client, whSteps *steps.OutgoingWebhookSteps) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("outgoing-webhook",
		func(ctx hatchet.Context, input port.OutgoingWebhookPayload) (port.WorkflowResult, error) {
			if err := whSteps.SendWebhook(ctx, input); err != nil {
				return port.WorkflowResult{}, err
			}
			return port.WorkflowResult{Success: true, Message: "sent"}, nil
		},
		hatchet.WithExecutionTimeout(15*time.Second),
		hatchet.WithRetries(5),
		hatchet.WithRetryBackoff(1.0, 60),
	)
}
