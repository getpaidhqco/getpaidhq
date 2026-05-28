package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	hatchetwf "getpaidhq/internal/adapter/hatchet/workflows"
	temporalwf "getpaidhq/internal/adapter/temporal/workflows"
)

// CLAUDE.md elevates engine parity to a load-bearing invariant: the Hatchet
// and Temporal adapters "provide parity over the same workflow surface", with
// step/activity files that mirror each other 1:1 and deterministic addressing
// for every per-aggregate workflow. These guards lock the per-(org, aggregate)
// identity so a change to one engine's id/key format that isn't mirrored in
// the other fails CI — which matters because only one of the two engines is
// covered by an in-process test framework (Temporal's testsuite), and any
// divergence would let Hatchet drift undetected.

func TestEngineParity_SubscriptionIdentityMatches(t *testing.T) {
	cases := []struct {
		org, sub string
	}{
		{"org_1", "sub_1"},
		{"acme", "sub_abc123"},
		{"o", "s"},
	}
	for _, c := range cases {
		t.Run(c.org+"/"+c.sub, func(t *testing.T) {
			assert.Equal(t,
				hatchetwf.SubscriptionRunKey(c.org, c.sub),
				temporalwf.SubscriptionWorkflowID(c.org, c.sub),
				"per-subscription addressing must be identical across engines")
		})
	}
}

func TestEngineParity_BillingCycleIdentityMatches(t *testing.T) {
	cases := []struct {
		org, sub string
		cycle    int
	}{
		{"org_1", "sub_1", 0},
		{"org_1", "sub_1", 7},
		{"acme", "sub_abc123", 42},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			assert.Equal(t,
				hatchetwf.BillingRunKey(c.org, c.sub, c.cycle),
				temporalwf.BillingCycleWorkflowID(c.org, c.sub, c.cycle),
				"billing-cycle ids must match across engines")
		})
	}
}

func TestEngineParity_ReminderIdentityMatches(t *testing.T) {
	at := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	assert.Equal(t,
		hatchetwf.ReminderRunKey("org_1", "sub_1", at),
		temporalwf.ReminderWorkflowID("org_1", "sub_1", at),
		"reminder ids must match across engines")
}

func TestEngineParity_DunningIdentityMatches(t *testing.T) {
	assert.Equal(t,
		hatchetwf.DunningRunKey("org_1", "dc_1"),
		temporalwf.DunningWorkflowID("org_1", "dc_1"),
		"dunning runner ids must match across engines")
	assert.Equal(t,
		hatchetwf.DunningAttemptRunKey("org_1", "dc_1", 3),
		temporalwf.DunningAttemptWorkflowID("org_1", "dc_1", 3),
		"dunning attempt ids must match across engines")
}

// The signal names are constants on the Temporal side and string literals on
// the Hatchet event-bridge side. Hatchet's SubscriptionEventBridge feeds the
// same strings into UpdateEventKey, so any rename here means a coordinated
// edit in both adapters and the service that publishes the topic — locking
// the literal values catches accidental drift.
func TestEngineParity_SignalNameContract(t *testing.T) {
	// Subscription signals.
	assert.Equal(t, "subscription.paused", temporalwf.SignalSubscriptionPaused)
	assert.Equal(t, "subscription.resumed", temporalwf.SignalSubscriptionResumed)
	assert.Equal(t, "subscription.cancelled", temporalwf.SignalSubscriptionCancelled)
	assert.Equal(t, "subscription.activated", temporalwf.SignalSubscriptionActivated)
	assert.Equal(t, "refresh-state", temporalwf.SignalRefreshState)
	assert.Equal(t, "cancel", temporalwf.SignalCancelRunner)

	// Dunning signals.
	assert.Equal(t, "dunning.pause", temporalwf.SignalDunningPause)
	assert.Equal(t, "dunning.resume", temporalwf.SignalDunningResume)
	assert.Equal(t, "dunning.cancel", temporalwf.SignalDunningCancel)
	assert.Equal(t, "dunning.payment_method_updated", temporalwf.SignalDunningPaymentMethodUpd)
}
