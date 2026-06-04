package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReminderConfig_RoundTrip(t *testing.T) {
	cfg := ReminderConfig{Enabled: true, Offsets: []time.Duration{168 * time.Hour, 24 * time.Hour}}
	raw, err := cfg.Marshal()
	require.NoError(t, err)

	got, err := ParseReminderConfig(raw)
	require.NoError(t, err)
	require.True(t, got.Enabled)
	require.Equal(t, []time.Duration{168 * time.Hour, 24 * time.Hour}, got.Offsets)
}

func TestParseReminderConfig_EmptyIsDefault(t *testing.T) {
	got, err := ParseReminderConfig("")
	require.NoError(t, err)
	require.Equal(t, DefaultReminderConfig(), got)
}
