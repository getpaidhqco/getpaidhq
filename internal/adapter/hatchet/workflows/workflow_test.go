package workflows

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
)

// ============================================================
// Dunning runner pure helpers
// ============================================================

func TestShouldUseImmediateRetries(t *testing.T) {
	t.Run("empty reason returns false", func(t *testing.T) {
		assert.False(t, shouldUseImmediateRetries("", []string{"rate_limit"}))
	})

	t.Run("matching failure type returns true", func(t *testing.T) {
		assert.True(t, shouldUseImmediateRetries("rate_limit", []string{"rate_limit", "network_error"}))
	})

	t.Run("non-matching failure type returns false", func(t *testing.T) {
		assert.False(t, shouldUseImmediateRetries("insufficient_funds", []string{"rate_limit", "network_error"}))
	})

	t.Run("single-element slice matches", func(t *testing.T) {
		assert.True(t, shouldUseImmediateRetries("network_error", []string{"network_error"}))
	})

	t.Run("empty failure types slice returns false", func(t *testing.T) {
		assert.False(t, shouldUseImmediateRetries("rate_limit", []string{}))
	})

	t.Run("exact match required — partial substring is false", func(t *testing.T) {
		assert.False(t, shouldUseImmediateRetries("rate_limited", []string{"rate_limit"}))
	})
}

func TestIsDunningTerminal(t *testing.T) {
	t.Run("recovered is terminal", func(t *testing.T) {
		assert.True(t, isDunningTerminal(domain.DunningStatusRecovered))
	})
	t.Run("failed is terminal", func(t *testing.T) {
		assert.True(t, isDunningTerminal(domain.DunningStatusFailed))
	})
	t.Run("cancelled is terminal", func(t *testing.T) {
		assert.True(t, isDunningTerminal(domain.DunningStatusCancelled))
	})
	t.Run("expired is terminal", func(t *testing.T) {
		assert.True(t, isDunningTerminal(domain.DunningStatusExpired))
	})

	t.Run("active is NOT terminal", func(t *testing.T) {
		assert.False(t, isDunningTerminal(domain.DunningStatusActive))
	})
	t.Run("paused is NOT terminal", func(t *testing.T) {
		assert.False(t, isDunningTerminal(domain.DunningStatusPaused))
	})
}

// ============================================================
// Subscription runner pure helpers
// ============================================================

func TestIsTerminalStatus(t *testing.T) {
	t.Run("cancelled is terminal", func(t *testing.T) {
		assert.True(t, isTerminalStatus(domain.SubscriptionStatusCancelled))
	})
	t.Run("expired is terminal", func(t *testing.T) {
		assert.True(t, isTerminalStatus(domain.SubscriptionStatusExpired))
	})
	t.Run("completed is terminal", func(t *testing.T) {
		assert.True(t, isTerminalStatus(domain.SubscriptionStatusCompleted))
	})

	t.Run("active is NOT terminal", func(t *testing.T) {
		assert.False(t, isTerminalStatus(domain.SubscriptionStatusActive))
	})
	t.Run("trial is NOT terminal", func(t *testing.T) {
		assert.False(t, isTerminalStatus(domain.SubscriptionStatusTrial))
	})
	t.Run("past_due is NOT terminal", func(t *testing.T) {
		assert.False(t, isTerminalStatus(domain.SubscriptionStatusPastDue))
	})
	t.Run("paused is NOT terminal", func(t *testing.T) {
		assert.False(t, isTerminalStatus(domain.SubscriptionStatusPaused))
	})
	t.Run("unpaid is NOT terminal", func(t *testing.T) {
		assert.False(t, isTerminalStatus(domain.SubscriptionStatusUnpaid))
	})
	t.Run("error is NOT terminal", func(t *testing.T) {
		assert.False(t, isTerminalStatus(domain.SubscriptionStatusError))
	})
}

func TestContainsKey(t *testing.T) {
	t.Run("detects present key", func(t *testing.T) {
		assert.True(t, containsKey([]string{"a", "b", "c"}, "b"))
	})

	t.Run("returns false for absent key", func(t *testing.T) {
		assert.False(t, containsKey([]string{"a", "b"}, "c"))
	})

	t.Run("empty slice returns false", func(t *testing.T) {
		assert.False(t, containsKey([]string{}, "a"))
	})

	t.Run("nil slice returns false", func(t *testing.T) {
		assert.False(t, containsKey(nil, "a"))
	})

	t.Run("exact match required", func(t *testing.T) {
		assert.False(t, containsKey([]string{"abc"}, "ab"))
	})

	t.Run("first element matches", func(t *testing.T) {
		assert.True(t, containsKey([]string{"x", "y"}, "x"))
	})

	t.Run("last element matches", func(t *testing.T) {
		assert.True(t, containsKey([]string{"x", "y"}, "y"))
	})
}

// ============================================================
// Wait result helpers
// ============================================================

func TestWaitedKeys_NilSafety(t *testing.T) {
	require.Nil(t, waitedKeys(nil))
}

// ============================================================
// Dunning key generation — pin format contracts
// ============================================================

func TestDunningRunKey_Format(t *testing.T) {
	assert.Equal(t, "dunning_org_1_dc_5", DunningRunKey("org_1", "dc_5"))
}

func TestDunningRunKey_IdempotentPerCampaign(t *testing.T) {
	// Same org+ campaign always produces the same key (Hatchet WithRunKey contract).
	a := DunningRunKey("org_X", "camp_X")
	b := DunningRunKey("org_X", "camp_X")
	assert.Equal(t, a, b)
}

func TestDunningRunKey_DifferentCampaignsDiffer(t *testing.T) {
	a := DunningRunKey("org_1", "camp_a")
	b := DunningRunKey("org_1", "camp_b")
	assert.NotEqual(t, a, b)
}

func TestDunningAttemptRunKey_Format(t *testing.T) {
	assert.Equal(t, "dunning_attempt_org_1_dc_5_3", DunningAttemptRunKey("org_1", "dc_5", 3))
}

func TestDunningAttemptRunKey_DifferentAttemptNumbersDiffer(t *testing.T) {
	a := DunningAttemptRunKey("org_1", "dc_1", 1)
	b := DunningAttemptRunKey("org_1", "dc_1", 2)
	assert.NotEqual(t, a, b)
}

func TestDunningSignalKey_Format(t *testing.T) {
	assert.Equal(t, "dunning_signal:dunning.pause:org_1:dc_1",
		DunningSignalKey("dunning.pause", "org_1", "dc_1"))
}

func TestDunningSignalKey_DifferentSignalsDiffer(t *testing.T) {
	pause := DunningSignalKey("dunning.pause", "org_1", "dc_1")
	cancel := DunningSignalKey("dunning.cancel", "org_1", "dc_1")
	assert.NotEqual(t, pause, cancel)
}

func TestDunningPaymentMethodUpdatedKey_Format(t *testing.T) {
	assert.Equal(t, "dunning_pm_updated:org_1:dc_1",
		DunningPaymentMethodUpdatedKey("org_1", "dc_1"))
}

// ============================================================
// Dunning runner input integrity
// ============================================================

func TestDunningRunnerInput_ZeroValueIsSafe(t *testing.T) {
	// The zero-value input should populate empty strings / zeroes
	// without panicking, because the runner reads these for logging/metadata.
	var in DunningRunnerInput
	assert.Equal(t, "", in.OrgId)
	assert.Equal(t, "", in.CampaignId)
	assert.Equal(t, int64(0), in.FailedAmount)
}

func TestDunningAttemptInput_Fields(t *testing.T) {
	in := DunningAttemptInput{
		OrgId:         "org_1",
		CampaignId:    "dc_1",
		AttemptNumber: 2,
		AttemptType:   domain.DunningAttemptTypeProgressive,
	}
	assert.NotEmpty(t, in.OrgId)
	assert.Equal(t, 2, in.AttemptNumber)
	assert.Equal(t, domain.DunningAttemptTypeProgressive, in.AttemptType)
}

// ============================================================
// Subscription key generation — extended format tests
// ============================================================

func TestSubscriptionRunKey_UniquenessAcrossDifferentIds(t *testing.T) {
	assert.NotEqual(t,
		SubscriptionRunKey("org_1", "sub_a"),
		SubscriptionRunKey("org_1", "sub_b"),
	)
	assert.NotEqual(t,
		SubscriptionRunKey("org_a", "sub_1"),
		SubscriptionRunKey("org_b", "sub_1"),
	)
}

func TestUpdateEventKey_AllKnownEvents(t *testing.T) {
	tests := []struct{ event, want string }{
		{"subscription.paused", "update:subscription.paused:org_1:sub_2"},
		{"subscription.resumed", "update:subscription.resumed:org_1:sub_2"},
		{"subscription.cancelled", "update:subscription.cancelled:org_1:sub_2"},
		{"subscription.activated", "update:subscription.activated:org_1:sub_2"},
		{"refresh-state", "update:refresh-state:org_1:sub_2"},
	}
	for _, tc := range tests {
		t.Run(tc.event, func(t *testing.T) {
			assert.Equal(t, tc.want, UpdateEventKey(tc.event, "org_1", "sub_2"))
		})
	}
}

func TestBillingRunKey_StableAcrossTime(t *testing.T) {
	// BillingRunKey does NOT include a timestamp — it should be stable.
	k1 := BillingRunKey("org_1", "sub_2", 0)
	k2 := BillingRunKey("org_1", "sub_2", 0)
	assert.Equal(t, k1, k2)
}

func TestBillingRunKey_CycleIncrementChangesKey(t *testing.T) {
	assert.NotEqual(t,
		BillingRunKey("org_1", "sub_2", 0),
		BillingRunKey("org_1", "sub_2", 1),
	)
}

func TestReminderRunKey_DailyGranularity(t *testing.T) {
	morning := time.Date(2026, time.May, 11, 8, 30, 0, 0, time.UTC)
	evening := time.Date(2026, time.May, 11, 20, 45, 0, 0, time.UTC)
	nextDay := time.Date(2026, time.May, 12, 8, 30, 0, 0, time.UTC)

	// Same day produces the same key (de-duplication).
	assert.Equal(t, ReminderRunKey("org_1", "sub_2", morning), ReminderRunKey("org_1", "sub_2", evening))
	// Different day produces a different key.
	assert.NotEqual(t, ReminderRunKey("org_1", "sub_2", morning), ReminderRunKey("org_1", "sub_2", nextDay))
}
