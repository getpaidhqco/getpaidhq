package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRetryPolicy_GetNextCharge pins the retry-spacing math:
//
//	nextCharge = RenewsAt + (RetryPeriod * interval) / (RetryAttempts - Retries)
//
// so each successive retry of the same campaign lands a larger gap from the
// renewal date, and the schedule terminates (zero time) once attempts are spent.
// A frozen base keeps it deterministic; expected offsets are written as plain
// duration literals rather than re-deriving the implementation's division.
func TestRetryPolicy_GetNextCharge(t *testing.T) {
	t.Parallel()

	base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	// 14 day-units spread over 5 attempts → total window 336h, divided by the
	// number of attempts still remaining at each retry.
	daily := RetryPolicy{RetryAttempts: 5, RetryInterval: RetryIntervalDay, RetryPeriod: 14}

	tests := []struct {
		name       string
		policy     RetryPolicy
		retries    int
		wantOffset time.Duration // gap from base; ignored when wantZero
		wantZero   bool
	}{
		{name: "daily retry 1 of 5 (336h/5)", policy: daily, retries: 0, wantOffset: 67*time.Hour + 12*time.Minute},
		{name: "daily retry 2 of 5 (336h/4)", policy: daily, retries: 1, wantOffset: 84 * time.Hour},
		{name: "daily retry 3 of 5 (336h/3)", policy: daily, retries: 2, wantOffset: 112 * time.Hour},
		{name: "daily retry 4 of 5 (336h/2)", policy: daily, retries: 3, wantOffset: 168 * time.Hour},
		{name: "daily retry 5 of 5 (336h/1)", policy: daily, retries: 4, wantOffset: 336 * time.Hour},
		{name: "attempts exhausted returns zero", policy: daily, retries: 5, wantZero: true},
		{name: "beyond exhausted returns zero (no divide-by-zero)", policy: daily, retries: 6, wantZero: true},
		{
			name:       "hour interval",
			policy:     RetryPolicy{RetryAttempts: 4, RetryInterval: RetryIntervalHour, RetryPeriod: 8},
			retries:    0,
			wantOffset: 2 * time.Hour, // 8h / 4
		},
		{
			name:       "week interval",
			policy:     RetryPolicy{RetryAttempts: 2, RetryInterval: RetryIntervalWeek, RetryPeriod: 2},
			retries:    0,
			wantOffset: 168 * time.Hour, // (2 * 168h) / 2
		},
		{
			name:       "minute interval",
			policy:     RetryPolicy{RetryAttempts: 3, RetryInterval: RetryIntervalMinute, RetryPeriod: 9},
			retries:    0,
			wantOffset: 3 * time.Minute, // 9m / 3
		},
		{
			name:       "unknown interval falls back to daily",
			policy:     RetryPolicy{RetryAttempts: 2, RetryInterval: RetryInterval("year"), RetryPeriod: 2},
			retries:    0,
			wantOffset: 24 * time.Hour, // (2 * 24h) / 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sub := Subscription{RenewsAt: base, Retries: tt.retries}
			got := tt.policy.GetNextCharge(sub)

			if tt.wantZero {
				assert.True(t, got.IsZero(), "expected zero time when attempts are exhausted, got %s", got)
				return
			}
			assert.Equal(t, base.Add(tt.wantOffset), got)
		})
	}
}
