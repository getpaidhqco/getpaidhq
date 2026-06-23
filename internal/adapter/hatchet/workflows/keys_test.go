package workflows

import (
	"getpaidhq/internal/core/domain"
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

func TestPaymentSuccessRunKey(t *testing.T) {
	if got := PaymentSuccessRunKey("org_1", "ord_2", domain.Paystack, "psp_3"); got != "payment_success_org_1_ord_2_Paystack_psp_3" {
		t.Errorf("PaymentSuccessRunKey: got %q", got)
	}
	if PaymentSuccessRunKey("org_1", "ord_2", domain.Paystack, "psp_3") == PaymentSuccessRunKey("org_1", "ord_2", domain.Paystack, "psp_4") {
		t.Error("PaymentSuccessRunKey must differ by PSP payment identity")
	}
}

func TestPaymentRefundedRunKey(t *testing.T) {
	if got := PaymentRefundedRunKey("org_1", "ord_2", domain.CheckoutDotCom, "pay_3"); got != "payment_refunded_org_1_ord_2_CheckoutDotCom_pay_3" {
		t.Errorf("PaymentRefundedRunKey: got %q", got)
	}
	if PaymentRefundedRunKey("org_1", "ord_2", domain.CheckoutDotCom, "pay_3") == PaymentRefundedRunKey("org_1", "ord_3", domain.CheckoutDotCom, "pay_3") {
		t.Error("PaymentRefundedRunKey must differ by order")
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

func TestOrgBillingRunKey(t *testing.T) {
	bucket := time.Date(2026, time.May, 11, 12, 35, 0, 0, time.UTC)
	if got := OrgBillingRunKey("org_1", bucket); got != "orgbilling_org_1_202605111235" {
		t.Errorf("OrgBillingRunKey: got %q", got)
	}
}

func TestSweepCadence(t *testing.T) {
	cases := []struct {
		in       time.Duration
		wantTick time.Duration
		wantCron string
	}{
		{5 * time.Minute, 5 * time.Minute, "*/5 * * * *"},
		{0, time.Minute, "*/1 * * * *"},                    // unset/zero clamps up
		{20 * time.Second, time.Minute, "*/1 * * * *"},     // sub-minute clamps up
		{90 * time.Second, 2 * time.Minute, "*/2 * * * *"}, // rounds to whole minutes
		{time.Hour, time.Hour, "0 * * * *"},
		{3 * time.Hour, time.Hour, "0 * * * *"}, // >1h clamps down
	}
	for _, c := range cases {
		tick, cron := SweepCadence(c.in)
		if tick != c.wantTick || cron != c.wantCron {
			t.Errorf("SweepCadence(%v): got (%v, %q), want (%v, %q)", c.in, tick, cron, c.wantTick, c.wantCron)
		}
	}
}
