package workflows

import (
	"testing"
	"time"
)

// Pin the workflow-id and signal-name formats. The engine starts/signals
// workflows using these helpers; the runner observes signals on the same
// channel names. Drift between the two sides would silently break
// update/cancel/webhook delivery, so freeze the formatting here.
//
// These intentionally match the Hatchet keys_test.go so a future refactor
// extracting both into a shared helper has a stable contract to merge to.

func TestSubscriptionWorkflowID(t *testing.T) {
	if got := SubscriptionWorkflowID("org_1", "sub_2"); got != "sub_org_1_sub_2" {
		t.Errorf("SubscriptionWorkflowID: got %q", got)
	}
}

func TestReminderWorkflowID(t *testing.T) {
	at := time.Date(2026, time.May, 11, 12, 0, 0, 0, time.UTC)
	if got := ReminderWorkflowID("org_1", "sub_2", at); got != "reminder_org_1_sub_2_20260511" {
		t.Errorf("ReminderWorkflowID: got %q", got)
	}
}

func TestBillingCycleWorkflowID(t *testing.T) {
	if got := BillingCycleWorkflowID("org_1", "sub_2", 7); got != "billing_org_1_sub_2_7" {
		t.Errorf("BillingCycleWorkflowID: got %q", got)
	}
}

func TestWebhookSignalName(t *testing.T) {
	if got := WebhookSignalName("org_1", "sub_2"); got != "webhook:org_1:sub_2" {
		t.Errorf("WebhookSignalName: got %q", got)
	}
}

func TestDunningWorkflowID(t *testing.T) {
	if got := DunningWorkflowID("org_1", "camp_2"); got != "dunning_org_1_camp_2" {
		t.Errorf("DunningWorkflowID: got %q", got)
	}
}

func TestDunningAttemptWorkflowID(t *testing.T) {
	if got := DunningAttemptWorkflowID("org_1", "camp_2", 3); got != "dunning_attempt_org_1_camp_2_3" {
		t.Errorf("DunningAttemptWorkflowID: got %q", got)
	}
}

func TestSignalNames(t *testing.T) {
	// The signal names are wire constants — bumping them needs a coordinated
	// release.
	cases := map[string]string{
		SignalSubscriptionPaused:      "subscription.paused",
		SignalSubscriptionResumed:     "subscription.resumed",
		SignalSubscriptionCancelled:   "subscription.cancelled",
		SignalSubscriptionActivated:   "subscription.activated",
		SignalRefreshState:            "refresh-state",
		SignalCancelRunner:            "cancel",
		SignalDunningPause:            "dunning.pause",
		SignalDunningResume:           "dunning.resume",
		SignalDunningCancel:           "dunning.cancel",
		SignalDunningPaymentMethodUpd: "dunning.payment_method_updated",
	}
	for actual, expected := range cases {
		if actual != expected {
			t.Errorf("signal mismatch: %q != %q", actual, expected)
		}
	}
}
