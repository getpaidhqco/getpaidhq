package domain

import (
	"encoding/json"
	"time"
)

// Setting coordinates for the per-org renewal-reminder config.
const (
	ReminderConfigSettingParent = "billing"
	ReminderConfigSettingId     = "renewal_reminders"
)

// ReminderConfig is the resolved per-tenant renewal-reminder policy.
type ReminderConfig struct {
	Enabled bool            `json:"enabled"`
	Offsets []time.Duration `json:"-"` // lead times before renewal, e.g. 168h, 24h
}

// reminderConfigJSON is the persisted shape (durations as human strings like
// "168h", not int64 nanoseconds) for a readable/editable setting value.
type reminderConfigJSON struct {
	Enabled bool     `json:"enabled"`
	Offsets []string `json:"offsets"`
}

// DefaultReminderConfig is the fallback when an org has no reminder setting:
// one reminder 7 days before renewal. Tenants override (incl. disable) via the
// reminder-config endpoint. Mirrors DefaultDunningConfig()'s role.
func DefaultReminderConfig() ReminderConfig {
	return ReminderConfig{Enabled: true, Offsets: []time.Duration{7 * 24 * time.Hour}}
}

// Marshal renders the config to the persisted JSON string (durations as strings).
func (c ReminderConfig) Marshal() (string, error) {
	dto := reminderConfigJSON{Enabled: c.Enabled}
	for _, d := range c.Offsets {
		dto.Offsets = append(dto.Offsets, d.String())
	}
	b, err := json.Marshal(dto)
	return string(b), err
}

// ParseReminderConfig parses a persisted value; empty input returns the default.
func ParseReminderConfig(raw string) (ReminderConfig, error) {
	if raw == "" {
		return DefaultReminderConfig(), nil
	}
	var dto reminderConfigJSON
	if err := json.Unmarshal([]byte(raw), &dto); err != nil {
		return ReminderConfig{}, err
	}
	cfg := ReminderConfig{Enabled: dto.Enabled}
	for _, s := range dto.Offsets {
		d, err := time.ParseDuration(s)
		if err != nil {
			return ReminderConfig{}, err
		}
		cfg.Offsets = append(cfg.Offsets, d)
	}
	return cfg, nil
}
