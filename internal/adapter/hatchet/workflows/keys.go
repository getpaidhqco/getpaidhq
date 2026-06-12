package workflows

import (
	"fmt"
	"strconv"
	"time"
)

// Event-key and run-key conventions for the Hatchet adapter.
//
// These are the contract between the Engine port (which pushes events) and the
// durable subscription runner (which awaits them). Both sides must use the
// same helper — never inline a string.

// SubscriptionRunKey is the deterministic run key for the per-subscription
// durable task. Using WithRunKey(...) makes StartSubscriptionWorkflow idempotent.
func SubscriptionRunKey(orgId, subscriptionId string) string {
	return fmt.Sprintf("sub_%s_%s", orgId, subscriptionId)
}

// UpdateEventKey carries a domain.Subscription state change to the durable runner.
// updateName values: "subscription.paused", "subscription.resumed",
// "subscription.cancelled", "subscription.activated", "refresh-state".
func UpdateEventKey(updateName, orgId, subscriptionId string) string {
	return fmt.Sprintf("update:%s:%s:%s", updateName, orgId, subscriptionId)
}

// CancelEventKey signals a graceful exit to the durable runner.
func CancelEventKey(orgId, subscriptionId string) string {
	return fmt.Sprintf("cancel:%s:%s", orgId, subscriptionId)
}

// WebhookEventKey carries a domain.ChargeResult that resolves a Pending payment.
func WebhookEventKey(orgId, subscriptionId string) string {
	return fmt.Sprintf("webhook:%s:%s", orgId, subscriptionId)
}

// ReminderRunKey de-duplicates reminder spawns within a billing cycle.
func ReminderRunKey(orgId, subscriptionId string, reminderAt time.Time) string {
	return fmt.Sprintf("reminder_%s_%s_%s", orgId, subscriptionId, reminderAt.Format("20060102"))
}

// BillingRunKey de-duplicates billing-cycle spawns within an iteration of the
// subscription runner. Includes the cycle index so successive iterations
// produce new keys.
func BillingRunKey(orgId, subscriptionId string, cycle int) string {
	return fmt.Sprintf("billing_%s_%s_%s", orgId, subscriptionId, strconv.Itoa(cycle))
}

// OrgBillingRunKey dedups the per-org billing fan-out within a single sweep
// bucket (the timestamp truncated to the SweepCadence interval), so an
// accidental double-sweep in the same bucket doesn't double-spawn an org's
// billing run. Minute precision in the format so distinct buckets get
// distinct keys.
func OrgBillingRunKey(orgId string, bucket time.Time) string {
	return fmt.Sprintf("orgbilling_%s_%s", orgId, bucket.UTC().Format("200601021504"))
}

// ReminderStageRunKey dedups a renewal reminder to exactly once per
// (subscription, cycle, offset-stage). The sweep may re-spawn this every tick
// the sub is inside the stage window; identical keys collapse (USE_EXISTING),
// so the reminder sends once per stage per cycle and self-heals across missed
// ticks. `cycle` is CyclesProcessed; the offset label distinguishes stages
// (e.g. "168h" vs "24h").
func ReminderStageRunKey(orgId, subscriptionId string, cycle int, offset time.Duration) string {
	return fmt.Sprintf("reminder_%s_%s_%s_%s", orgId, subscriptionId, strconv.Itoa(cycle), offset.String())
}
