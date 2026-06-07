package workflows

import (
	"fmt"
	"strconv"

	"getpaidhq/internal/core/domain"
)

// DunningRunKey is the deterministic run key for the per-campaign dunning
// durable runner. Makes Engine.StartDunningWorkflow idempotent.
func DunningRunKey(orgId, campaignId string) string {
	return fmt.Sprintf("dunning_%s_%s", orgId, campaignId)
}

// DunningAttemptRunKey de-duplicates per-attempt DAG spawns within a campaign.
// The attempt type is part of the key because the immediate and progressive
// phases both number their attempts 1..N — without it, progressive attempt #1
// would collide with immediate attempt #1 and Hatchet would return the cached
// immediate result instead of running a real progressive charge.
func DunningAttemptRunKey(orgId, campaignId string, attemptType domain.DunningAttemptType, attemptNumber int) string {
	return fmt.Sprintf("dunning_attempt_%s_%s_%s_%s", orgId, campaignId, string(attemptType), strconv.Itoa(attemptNumber))
}

// DunningCommunicationRunKey de-duplicates per-attempt communication child
// spawns. Communications are only sent in the progressive phase, so the attempt
// number alone is unique.
func DunningCommunicationRunKey(orgId, campaignId string, attemptNumber int) string {
	return fmt.Sprintf("dunning_comm_%s_%s_%s", orgId, campaignId, strconv.Itoa(attemptNumber))
}

// DunningResultRunKey de-duplicates per-attempt result-application child spawns.
// Keyed by attempt type for the same reason as DunningAttemptRunKey.
func DunningResultRunKey(orgId, campaignId string, attemptType domain.DunningAttemptType, attemptNumber int) string {
	return fmt.Sprintf("dunning_result_%s_%s_%s_%s", orgId, campaignId, string(attemptType), strconv.Itoa(attemptNumber))
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
