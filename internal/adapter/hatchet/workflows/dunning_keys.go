package workflows

import (
	"fmt"
	"strconv"
)

// DunningRunKey is the deterministic run key for the per-campaign dunning
// durable runner. Makes Engine.StartDunningWorkflow idempotent.
func DunningRunKey(orgId, campaignId string) string {
	return fmt.Sprintf("dunning_%s_%s", orgId, campaignId)
}

// DunningAttemptRunKey de-duplicates per-attempt DAG spawns within a campaign.
func DunningAttemptRunKey(orgId, campaignId string, attemptNumber int) string {
	return fmt.Sprintf("dunning_attempt_%s_%s_%s", orgId, campaignId, strconv.Itoa(attemptNumber))
}

// DunningSignalKey carries a control signal (pause/resume/cancel/refresh) to
// the dunning durable runner.
func DunningSignalKey(signal, orgId, campaignId string) string {
	return fmt.Sprintf("dunning_signal:%s:%s:%s", signal, orgId, campaignId)
}

// DunningPaymentMethodUpdatedKey carries a payment-method-updated trigger so
// the runner can run an immediate retry attempt.
func DunningPaymentMethodUpdatedKey(orgId, campaignId string) string {
	return fmt.Sprintf("dunning_pm_updated:%s:%s", orgId, campaignId)
}
