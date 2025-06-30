package interfaces

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
)

// DunningService defines the interface for dunning service operations
type DunningService interface {
	// Campaign operations
	CreateCampaign(ctx context.Context, input CreateDunningCampaignInput) (dunning.DunningCampaign, error)
	FindCampaignById(ctx context.Context, orgId string, id string) (dunning.DunningCampaign, error)
	ListCampaigns(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error)
	ListCampaignsBySubscription(ctx context.Context, orgId string, subscriptionId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error)
	ListCampaignsByCustomer(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error)
	PauseCampaign(ctx context.Context, input PauseDunningCampaignInput) (dunning.DunningCampaign, error)
	ResumeCampaign(ctx context.Context, input ResumeDunningCampaignInput) (dunning.DunningCampaign, error)
	CancelCampaign(ctx context.Context, input CancelDunningCampaignInput) (dunning.DunningCampaign, error)
	UpdateCampaign(ctx context.Context, orgId string, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error)

	// Attempt operations
	ListAttemptsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningAttempt, int, error)
	TriggerManualAttempt(ctx context.Context, input TriggerManualAttemptInput) (dunning.DunningAttempt, error)

	// Communication operations
	ListCommunicationsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningCommunication, int, error)

	// Token operations
	CreatePaymentUpdateToken(ctx context.Context, input CreatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error)
	VerifyPaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error)
	ActivatePaymentUpdateToken(ctx context.Context, input ActivatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error)
	RevokePaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error)

	// Configuration operations
	CreateConfiguration(ctx context.Context, input CreateDunningConfigurationInput) (dunning.DunningConfiguration, error)
	GetConfiguration(ctx context.Context, orgId string, id string) (dunning.DunningConfiguration, error)
	ListConfigurations(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningConfiguration, int, error)
	UpdateConfiguration(ctx context.Context, input UpdateDunningConfigurationInput) (dunning.DunningConfiguration, error)

	// Customer dunning history operations
	GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (dunning.CustomerDunningHistory, error)
}

// DunningOrchestrationService defines the interface for dunning workflow operations
type DunningOrchestrationService interface {
	DunningService

	// Workflow operations
	StartDunningWorkflow(ctx context.Context, input StartDunningWorkflowInput) (dunning.DunningCampaign, error)
	HandlePaymentMethodUpdated(ctx context.Context, input PaymentMethodUpdatedInput) error
	HandleSubscriptionStateChanged(ctx context.Context, input SubscriptionStateChangedInput) error
	HandleDunningAttemptResult(ctx context.Context, input DunningAttemptResultInput) (dunning.DunningCampaign, error)
}

// Input types for dunning service operations

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

// TriggerManualAttemptInput defines the input for triggering a manual payment attempt
type TriggerManualAttemptInput struct {
	OrgId           string `json:"org_id"`
	CampaignId      string `json:"campaign_id"`
	PaymentMethodId string `json:"payment_method_id,omitempty"`
	TriggeredBy     string `json:"triggered_by,omitempty"`
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
