package dto

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
)

// CreateDunningCampaignInput defines the input for creating a dunning campaign
type CreateDunningCampaignInput struct {
	OrgId                string            `json:"org_id"`
	SubscriptionId       string            `json:"subscription_id"`
	CustomerId           string            `json:"customer_id"`
	FailedAmount         int               `json:"failed_amount"`
	Currency             string            `json:"currency"`
	InitialFailureReason string            `json:"initial_failure_reason,omitempty"`
	ParentWorkflowId     string            `json:"parent_workflow_id,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

// PauseDunningCampaignInput defines the input for pausing a dunning campaign
type PauseDunningCampaignInput struct {
	OrgId  string `json:"org_id"`
	Id     string `json:"id"`
	Reason string `json:"reason,omitempty"`
}

// ResumeDunningCampaignInput defines the input for resuming a dunning campaign
type ResumeDunningCampaignInput struct {
	OrgId  string `json:"org_id"`
	Id     string `json:"id"`
	Reason string `json:"reason,omitempty"`
}

// CancelDunningCampaignInput defines the input for cancelling a dunning campaign
type CancelDunningCampaignInput struct {
	OrgId  string `json:"org_id"`
	Id     string `json:"id"`
	Reason string `json:"reason,omitempty"`
}

// TriggerAttemptInput defines the input for triggering a manual payment attempt
type TriggerAttemptInput struct {
	OrgId           string                     `json:"org_id"`
	Type            dunning.DunningAttemptType `json:"type"`
	CampaignId      string                     `json:"campaign_id"`
	PaymentMethodId string                     `json:"payment_method_id,omitempty"`
	TriggeredBy     string                     `json:"triggered_by,omitempty"`
}

// CreatePaymentUpdateTokenInput defines the input for creating a payment update token
type CreatePaymentUpdateTokenInput struct {
	OrgId             string          `json:"org_id"`
	SubscriptionId    string          `json:"subscription_id"`
	CustomerId        string          `json:"customer_id"`
	DunningCampaignId string          `json:"dunning_campaign_id,omitempty"`
	MaxUses           int             `json:"max_uses,omitempty"`
	ExpiryHours       int             `json:"expiry_hours,omitempty"`
	AllowedActions    map[string]bool `json:"allowed_actions,omitempty"`
	AdminGenerated    bool            `json:"admin_generated,omitempty"`
	AdminUserId       string          `json:"admin_user_id,omitempty"`
	AdminReason       string          `json:"admin_reason,omitempty"`
	AdminNotes        string          `json:"admin_notes,omitempty"`
	CreatedBy         string          `json:"created_by"`
}

// ActivatePaymentUpdateTokenInput defines the input for activating a payment update token
type ActivatePaymentUpdateTokenInput struct {
	OrgId     string `json:"org_id"`
	TokenId   string `json:"token_id"`
	IpAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// CreateDunningConfigurationInput defines the input for creating a dunning configuration
type CreateDunningConfigurationInput struct {
	OrgId            string                     `json:"org_id"`
	Name             string                     `json:"name"`
	Description      string                     `json:"description,omitempty"`
	Priority         int                        `json:"priority,omitempty"`
	AppliesTo        dunning.DunningConfigScope `json:"applies_to"`
	TargetRules      map[string]interface{}     `json:"target_rules,omitempty"`
	Config           dunning.DunningConfig      `json:"config"`
	IsAbTest         bool                       `json:"is_ab_test,omitempty"`
	AbTestPercentage float64                    `json:"ab_test_percentage,omitempty"`
	CreatedBy        string                     `json:"created_by,omitempty"`
}

// UpdateDunningConfigurationInput defines the input for updating a dunning configuration
type UpdateDunningConfigurationInput struct {
	OrgId            string                     `json:"org_id"`
	Id               string                     `json:"id"`
	Name             string                     `json:"name,omitempty"`
	Description      string                     `json:"description,omitempty"`
	Priority         int                        `json:"priority,omitempty"`
	AppliesTo        dunning.DunningConfigScope `json:"applies_to,omitempty"`
	TargetRules      map[string]interface{}     `json:"target_rules,omitempty"`
	Config           dunning.DunningConfig      `json:"config,omitempty"`
	Status           dunning.ConfigStatus       `json:"status,omitempty"`
	IsAbTest         bool                       `json:"is_ab_test,omitempty"`
	AbTestPercentage float64                    `json:"ab_test_percentage,omitempty"`
}

// StartDunningWorkflowInput defines the input for starting a dunning workflow
type StartDunningWorkflowInput struct {
	OrgId                string                `json:"org_id"`
	SubscriptionId       string                `json:"subscription_id"`
	CustomerId           string                `json:"customer_id"`
	FailedAmount         int                   `json:"failed_amount"`
	Currency             string                `json:"currency"`
	InitialFailureReason string                `json:"initial_failure_reason,omitempty"`
	ParentWorkflowId     string                `json:"parent_workflow_id,omitempty"`
	PaymentResult        payments.ChargeResult `json:"payment_result"`
	Metadata             map[string]string     `json:"metadata,omitempty"`
}

// PaymentMethodUpdatedInput defines the input for handling a payment method update
type PaymentMethodUpdatedInput struct {
	OrgId             string `json:"org_id"`
	SubscriptionId    string `json:"subscription_id"`
	CustomerId        string `json:"customer_id"`
	PaymentMethodId   string `json:"payment_method_id"`
	DunningCampaignId string `json:"dunning_campaign_id,omitempty"`
}

// SubscriptionStateChangedInput defines the input for handling a subscription state change
type SubscriptionStateChangedInput struct {
	OrgId             string                      `json:"org_id"`
	SubscriptionId    string                      `json:"subscription_id"`
	OldStatus         entities.SubscriptionStatus `json:"old_status"`
	NewStatus         entities.SubscriptionStatus `json:"new_status"`
	DunningCampaignId string                      `json:"dunning_campaign_id,omitempty"`
}

// DunningAttemptResultInput defines the input for handling a dunning attempt result
type DunningAttemptResultInput struct {
	OrgId         string                `json:"org_id"`
	CampaignId    string                `json:"campaign_id"`
	AttemptId     string                `json:"attempt_id"`
	Success       bool                  `json:"success"`
	PaymentResult payments.ChargeResult `json:"payment_result"`
}

type HandleChargeResultResponse struct {
	Subscription entities.Subscription   `json:"subscription"`
	Campaign     dunning.DunningCampaign `json:"campaign"`
	Attempt      dunning.DunningAttempt  `json:"attempt"`
}
