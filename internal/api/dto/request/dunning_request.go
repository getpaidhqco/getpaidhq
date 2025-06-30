package request

import (
	"payloop/internal/domain/entities/dunning"
)

// VerifyPaymentTokenRequest represents a request to verify a payment token
type VerifyPaymentTokenRequest struct {
	TokenID string `json:"token_id" binding:"required"`
}

// ActivatePaymentTokenRequest represents a request to activate a payment token
type ActivatePaymentTokenRequest struct {
	TokenID string `json:"token_id" binding:"required"`
}

// CreatePaymentTokenRequest represents a request to create a payment token
type CreatePaymentTokenRequest struct {
	SubscriptionID string            `json:"subscription_id" binding:"required"`
	MaxUses        int               `json:"max_uses,omitempty"`
	ExpiryHours    int               `json:"expiry_hours,omitempty"`
	AllowedActions map[string]bool   `json:"allowed_actions,omitempty"`
	AdminReason    string            `json:"admin_reason,omitempty"`
	AdminNotes     string            `json:"admin_notes,omitempty"`
}

// UpdateDunningCampaignRequest represents a request to update a dunning campaign
type UpdateDunningCampaignRequest struct {
	Status string `json:"status" binding:"required,oneof=active paused cancelled"`
	Reason string `json:"reason,omitempty"`
}

// CreateDunningConfigurationRequest represents a request to create a dunning configuration
type CreateDunningConfigurationRequest struct {
	Name             string                   `json:"name" binding:"required"`
	Description      string                   `json:"description,omitempty"`
	Priority         int                      `json:"priority,omitempty"`
	AppliesTo        dunning.DunningConfigScope `json:"applies_to" binding:"required"`
	TargetRules      map[string]interface{}   `json:"target_rules,omitempty"`
	Config           dunning.DunningConfig    `json:"config" binding:"required"`
	IsAbTest         bool                     `json:"is_ab_test,omitempty"`
	AbTestPercentage float64                  `json:"ab_test_percentage,omitempty"`
}

// UpdateDunningConfigurationRequest represents a request to update a dunning configuration
type UpdateDunningConfigurationRequest struct {
	Name             string                   `json:"name,omitempty"`
	Description      string                   `json:"description,omitempty"`
	Priority         int                      `json:"priority,omitempty"`
	AppliesTo        dunning.DunningConfigScope `json:"applies_to,omitempty"`
	TargetRules      map[string]interface{}   `json:"target_rules,omitempty"`
	Config           dunning.DunningConfig    `json:"config,omitempty"`
	Status           dunning.ConfigStatus     `json:"status,omitempty"`
	IsAbTest         bool                     `json:"is_ab_test,omitempty"`
	AbTestPercentage float64                  `json:"ab_test_percentage,omitempty"`
}

// TriggerManualAttemptRequest represents a request to trigger a manual payment attempt
type TriggerManualAttemptRequest struct {
	PaymentMethodID string `json:"payment_method_id,omitempty"`
}