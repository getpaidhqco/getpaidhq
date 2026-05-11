package workflows

import (
	"testing"
	"time"
)

// Pin the key formats. The engine pushes events using these helpers; the
// durable runner subscribes using the same helpers. Drift between the two
// sides would silently break update/cancel/webhook delivery, so freeze
// the formatting here.

func TestSubscriptionRunKey(t *testing.T) {
	if got := SubscriptionRunKey("org_1", "sub_2"); got != "sub_org_1_sub_2" {
		t.Errorf("SubscriptionRunKey: got %q", got)
	}
}

func TestUpdateEventKey(t *testing.T) {
	if got := UpdateEventKey("subscription.paused", "org_1", "sub_2"); got != "update:subscription.paused:org_1:sub_2" {
		t.Errorf("UpdateEventKey: got %q", got)
	}
}

func TestCancelEventKey(t *testing.T) {
	if got := CancelEventKey("org_1", "sub_2"); got != "cancel:org_1:sub_2" {
		t.Errorf("CancelEventKey: got %q", got)
	}
}

func TestWebhookEventKey(t *testing.T) {
	if got := WebhookEventKey("org_1", "sub_2"); got != "webhook:org_1:sub_2" {
		t.Errorf("WebhookEventKey: got %q", got)
	}
}

func TestReminderRunKey(t *testing.T) {
	at := time.Date(2026, time.May, 11, 12, 0, 0, 0, time.UTC)
	if got := ReminderRunKey("org_1", "sub_2", at); got != "reminder_org_1_sub_2_20260511" {
		t.Errorf("ReminderRunKey: got %q", got)
	}
}

func TestBillingRunKey(t *testing.T) {
	if got := BillingRunKey("org_1", "sub_2", 7); got != "billing_org_1_sub_2_7" {
		t.Errorf("BillingRunKey: got %q", got)
	}
}
