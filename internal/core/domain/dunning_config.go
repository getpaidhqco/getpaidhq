package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DunningConfig is the typed retry + communication policy. Stored as JSON on
// DunningConfiguration.Config, snapshotted onto a campaign at start so
// mid-flight policy changes don't break a running campaign.
type DunningConfig struct {
	ImmediateRetries      ImmediateRetriesConfig   `json:"immediate_retries"`
	ProgressiveRetries    ProgressiveRetriesConfig `json:"progressive_retries"`
	EscalationRules       EscalationRulesConfig    `json:"escalation_rules"`
	CommunicationStrategy CommunicationStrategy    `json:"communication_strategy"`
	TokenSettings         TokenSettingsConfig      `json:"token_settings"`
}

type ImmediateRetriesConfig struct {
	Enabled      bool     `json:"enabled"`
	MaxAttempts  int      `json:"max_attempts"`
	Intervals    []string `json:"intervals"`
	FailureTypes []string `json:"failure_types"`
}

type ProgressiveRetriesConfig struct {
	Enabled      bool     `json:"enabled"`
	MaxAttempts  int      `json:"max_attempts"`
	Intervals    []string `json:"intervals"`
	FailureTypes []string `json:"failure_types"`
}

type EscalationRulesConfig struct {
	SuspendAfterAttempt int `json:"suspend_after_attempt"`
	FinalNoticeAttempt  int `json:"final_notice_attempt"`
	CancelAfterAttempt  int `json:"cancel_after_attempt"`
}

type CommunicationStrategy struct {
	Channels map[string]CommunicationChannelConfig `json:"channels"`
}

type CommunicationChannelConfig struct {
	Enabled           bool              `json:"enabled"`
	Templates         map[string]string `json:"templates"`
	StartAfterAttempt int               `json:"start_after_attempt"`
}

type TokenSettingsConfig struct {
	DefaultMaxUses     int `json:"default_max_uses"`
	DefaultExpiryHours int `json:"default_expiry_hours"`
}

// DefaultDunningConfig returns the fallback policy used when an org has no
// matching DunningConfiguration. Values match the original implementation.
func DefaultDunningConfig() DunningConfig {
	return DunningConfig{
		ImmediateRetries: ImmediateRetriesConfig{
			Enabled:      true,
			MaxAttempts:  3,
			Intervals:    []string{"2m", "10m", "30m"},
			FailureTypes: []string{"api_timeout", "gateway_error", "processing_error", "rate_limit", "network_error"},
		},
		ProgressiveRetries: ProgressiveRetriesConfig{
			Enabled:      true,
			MaxAttempts:  5,
			Intervals:    []string{"3d", "4d", "7d", "14d", "30d"},
			FailureTypes: []string{"card_declined", "insufficient_funds", "expired_card", "do_not_honor", "generic_decline"},
		},
		EscalationRules: EscalationRulesConfig{
			SuspendAfterAttempt: 3,
			FinalNoticeAttempt:  4,
			CancelAfterAttempt:  5,
		},
		CommunicationStrategy: CommunicationStrategy{
			Channels: map[string]CommunicationChannelConfig{
				"email": {
					Enabled: true,
					Templates: map[string]string{
						"attempt_1": "dunning_gentle_reminder",
						"attempt_2": "dunning_payment_failed",
						"attempt_3": "dunning_urgent_notice",
						"attempt_4": "dunning_final_notice",
					},
					StartAfterAttempt: 0,
				},
				"sms": {
					Enabled: true,
					Templates: map[string]string{
						"attempt_3": "dunning_sms_urgent",
						"attempt_4": "dunning_sms_final",
					},
					StartAfterAttempt: 3,
				},
			},
		},
		TokenSettings: TokenSettingsConfig{
			DefaultMaxUses:     5,
			DefaultExpiryHours: 72,
		},
	}
}

// ParseDuration is dunning's interval format: "2m", "30m", "3d", "4h", "14d".
// time.ParseDuration doesn't accept "d" so we parse the suffix ourselves.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid days duration %q: %w", s, err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
