package workflows

import (
	"fmt"
	"strconv"
	"time"

	"getpaidhq/internal/core/domain"
)

// Workflow-ID and signal-name conventions for the Temporal adapter.
//
// These mirror internal/adapter/hatchet/workflows/keys.go so the two adapters
// share identical addressing semantics: one per-aggregate workflow per
// (org, subscription) tuple, signals scoped to that same tuple.

// SubscriptionWorkflowID is the deterministic workflow id for the
// per-subscription long-running workflow. Combined with
// WorkflowIDReusePolicy=ALLOW_DUPLICATE+WorkflowIDConflictPolicy=FAIL on
// start, this makes StartSubscriptionWorkflow idempotent.
func SubscriptionWorkflowID(orgId, subscriptionId string) string {
	return fmt.Sprintf("sub_%s_%s", orgId, subscriptionId)
}

// ReminderWorkflowID de-duplicates a reminder to once per (sub, cycle, offset
// stage). Mirrors Hatchet's ReminderStageRunKey so both engines address the
// same reminder identically.
func ReminderWorkflowID(orgId, subscriptionId string, cycle int, offset time.Duration) string {
	return fmt.Sprintf("reminder_%s_%s_%d_%s", orgId, subscriptionId, cycle, offset.String())
}

// BillingCycleWorkflowID de-duplicates billing-cycle spawns within an
// iteration of the subscription runner. Includes the cycle index so
// successive iterations produce new ids.
func BillingCycleWorkflowID(orgId, subscriptionId string, cycle int) string {
	return fmt.Sprintf("billing_%s_%s_%s", orgId, subscriptionId, strconv.Itoa(cycle))
}

// Signal names sent to the per-subscription runner. Distinct from the
// per-subscription workflow-id; one workflow id receives multiple named
// signals on its dedicated channels.
const (
	SignalSubscriptionPaused    = "subscription.paused"
	SignalSubscriptionResumed   = "subscription.resumed"
	SignalSubscriptionCancelled = "subscription.cancelled"
	SignalSubscriptionActivated = "subscription.activated"
	SignalRefreshState          = "refresh-state"
	SignalCancelRunner          = "cancel"
)

// WebhookSignalName carries a domain.ChargeResult that resolves a Pending
// payment. Per-(org, sub) so concurrent runners do not collide.
func WebhookSignalName(orgId, subscriptionId string) string {
	return fmt.Sprintf("webhook:%s:%s", orgId, subscriptionId)
}

// ---- Dunning ----

// DunningWorkflowID is the deterministic workflow id for the per-campaign
// dunning runner.
func DunningWorkflowID(orgId, campaignId string) string {
	return fmt.Sprintf("dunning_%s_%s", orgId, campaignId)
}

// DunningAttemptWorkflowID de-duplicates per-attempt child spawns within a
// campaign. The attempt type is part of the id to mirror the Hatchet run key,
// where it is load-bearing: the immediate and progressive phases both number
// their attempts 1..N, and without the type the two phases' attempt #1 collide.
func DunningAttemptWorkflowID(orgId, campaignId string, attemptType domain.DunningAttemptType, attemptNumber int) string {
	return fmt.Sprintf("dunning_attempt_%s_%s_%s_%s", orgId, campaignId, string(attemptType), strconv.Itoa(attemptNumber))
}

// Dunning signal names — sent to the per-campaign runner.
const (
	SignalDunningPause            = "dunning.pause"
	SignalDunningResume           = "dunning.resume"
	SignalDunningCancel           = "dunning.cancel"
	SignalDunningPaymentMethodUpd = "dunning.payment_method_updated"
)
