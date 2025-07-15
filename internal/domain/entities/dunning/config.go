package dunning

import (
	"time"
)

// ChannelConfig represents the configuration for a communication channel
type ChannelConfig struct {
	Enabled           bool              `json:"enabled"`
	Templates         map[string]string `json:"templates"` // e.g. {"attempt_1": "dunning_gentle_reminder"}
	StartAfterAttempt int               `json:"start_after_attempt,omitempty"`
}

// DunningConfig represents the configuration for a dunning campaign
type DunningConfig struct {

	// Progressive retries for user-space errors
	ProgressiveRetries struct {
		Enabled      bool     `json:"enabled"`
		MaxAttempts  int      `json:"max_attempts"`
		Intervals    []string `json:"intervals"`     // e.g. ["3d", "4d", "7d", "14d", "30d"]
		FailureTypes []string `json:"failure_types"` // e.g. ["card_declined", "insufficient_funds", "expired_card"]
	} `json:"progressive_retries"`

	// Escalation rules
	EscalationRules struct {
		SuspendAfterAttempt int `json:"suspend_after_attempt"`
		FinalNoticeAttempt  int `json:"final_notice_attempt"`
		CancelAfterAttempt  int `json:"cancel_after_attempt"`
	} `json:"escalation_rules"`

	// Communication strategy
	CommunicationStrategy struct {
		Channels map[string]ChannelConfig `json:"channels"`
	} `json:"communication_strategy"`

	// Token settings
	TokenSettings struct {
		DefaultMaxUses     int `json:"default_max_uses"`
		DefaultExpiryHours int `json:"default_expiry_hours"`
	} `json:"token_settings"`
}

// DefaultDunningConfig returns a default dunning configuration
func DefaultDunningConfig() DunningConfig {
	config := DunningConfig{}

	// Progressive retries
	config.ProgressiveRetries.Enabled = true
	config.ProgressiveRetries.MaxAttempts = 5
	config.ProgressiveRetries.Intervals = []string{"3d", "4d", "7d", "14d", "30d"}
	config.ProgressiveRetries.FailureTypes = []string{
		"card_declined",
		"insufficient_funds",
		"expired_card",
		"do_not_honor",
		"generic_decline",
	}

	// Escalation rules
	config.EscalationRules.SuspendAfterAttempt = 3
	config.EscalationRules.FinalNoticeAttempt = 4
	config.EscalationRules.CancelAfterAttempt = 5

	// Communication strategy
	config.CommunicationStrategy.Channels = make(map[string]ChannelConfig)

	// Email channel
	config.CommunicationStrategy.Channels["email"] = ChannelConfig{
		Enabled: true,
		Templates: map[string]string{
			"attempt_1": "dunning_gentle_reminder",
			"attempt_2": "dunning_urgent_action",
			"attempt_3": "dunning_critical_notice",
			"attempt_4": "dunning_final_notice",
		},
		StartAfterAttempt: 0,
	}

	// SMS channel
	config.CommunicationStrategy.Channels["sms"] = ChannelConfig{
		Enabled: true,
		Templates: map[string]string{
			"attempt_3": "dunning_critical_sms",
			"attempt_4": "dunning_final_sms",
		},
		StartAfterAttempt: 3,
	}

	// Token settings
	config.TokenSettings.DefaultMaxUses = 5
	config.TokenSettings.DefaultExpiryHours = 72

	return config
}

// ParseDuration parses a duration string like "3d" or "10m" into a time.Duration
func ParseDuration(durationStr string) (time.Duration, error) {
	// Check for day format
	if len(durationStr) > 1 && durationStr[len(durationStr)-1] == 'd' {
		days, err := time.ParseDuration(durationStr[:len(durationStr)-1] + "h")
		if err != nil {
			return 0, err
		}
		return days * 24, nil
	}

	// Standard duration format
	return time.ParseDuration(durationStr)
}
