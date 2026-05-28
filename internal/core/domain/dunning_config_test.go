package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{name: "minutes", input: "2m", want: 2 * time.Minute},
		{name: "thirty minutes", input: "30m", want: 30 * time.Minute},
		{name: "hours", input: "4h", want: 4 * time.Hour},
		{name: "fractional hours", input: "1.5h", want: 90 * time.Minute},
		{name: "days (custom suffix)", input: "3d", want: 72 * time.Hour},
		{name: "fourteen days", input: "14d", want: 14 * 24 * time.Hour},
		{name: "zero days", input: "0d", want: 0},
		{name: "negative days", input: "-2d", want: -48 * time.Hour},
		{name: "surrounding whitespace trimmed", input: "  3d  ", want: 72 * time.Hour},
		{name: "empty string errors", input: "", wantErr: true},
		{name: "blank string errors", input: "   ", wantErr: true},
		{name: "non-numeric days errors", input: "xd", wantErr: true},
		{name: "garbage errors", input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDefaultDunningConfig_Invariants locks the structural assumptions the
// escalation engine relies on: the three escalation thresholds are strictly
// ordered, each retry tier has exactly one interval per attempt, and every
// interval string is parseable by ParseDuration (so a config typo can't silently
// stall a campaign at runtime).
func TestDefaultDunningConfig_Invariants(t *testing.T) {
	t.Parallel()

	cfg := DefaultDunningConfig()

	t.Run("escalation thresholds strictly ordered", func(t *testing.T) {
		t.Parallel()
		e := cfg.EscalationRules
		assert.Less(t, e.SuspendAfterAttempt, e.FinalNoticeAttempt, "suspend must come before final notice")
		assert.Less(t, e.FinalNoticeAttempt, e.CancelAfterAttempt, "final notice must come before cancel")
	})

	t.Run("one interval per attempt", func(t *testing.T) {
		t.Parallel()
		assert.Len(t, cfg.ImmediateRetries.Intervals, cfg.ImmediateRetries.MaxAttempts)
		assert.Len(t, cfg.ProgressiveRetries.Intervals, cfg.ProgressiveRetries.MaxAttempts)
	})

	t.Run("all intervals parse", func(t *testing.T) {
		t.Parallel()
		for _, iv := range append(append([]string{}, cfg.ImmediateRetries.Intervals...), cfg.ProgressiveRetries.Intervals...) {
			d, err := ParseDuration(iv)
			assert.NoErrorf(t, err, "interval %q should parse", iv)
			assert.Positivef(t, d, "interval %q should be positive", iv)
		}
	})
}
