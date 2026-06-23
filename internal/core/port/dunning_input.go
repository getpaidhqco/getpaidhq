package port

import "getpaidhq/internal/core/domain"

// CreateDunningCampaignInput is the input for DunningService.CreateCampaign.
type CreateDunningCampaignInput struct {
	OrgId                string
	SubscriptionId       string
	CustomerId           string
	FailedAmount         int64
	Currency             string
	InitialFailureReason string
	ParentWorkflowId     string
	ConfigSnapshot       map[string]any
	Metadata             map[string]string
}

// StartDunningWorkflowInput is the payload handed to DunningEngine.StartDunningWorkflow.
type StartDunningWorkflowInput struct {
	OrgId                string
	SubscriptionId       string
	CustomerId           string
	FailedAmount         int64
	Currency             string
	InitialFailureReason string
	ParentWorkflowId     string
	PaymentResult        domain.ChargeResult
	Metadata             map[string]string
}

// PauseDunningCampaignInput is the input for DunningService.PauseCampaign.
type PauseDunningCampaignInput struct {
	OrgId      string
	CampaignId string
	Reason     string
}

// ResumeDunningCampaignInput is the input for DunningService.ResumeCampaign.
type ResumeDunningCampaignInput struct {
	OrgId      string
	CampaignId string
	Reason     string
}

// CancelDunningCampaignInput is the input for DunningService.CancelCampaign.
type CancelDunningCampaignInput struct {
	OrgId      string
	CampaignId string
	Reason     string
}

// TriggerManualAttemptInput is the input for DunningService.TriggerManualAttempt.
type TriggerManualAttemptInput struct {
	OrgId           string
	CampaignId      string
	PaymentMethodId string
	TriggeredBy     string
}

// CreateDunningConfigurationInput is the input for DunningService.CreateConfiguration.
type CreateDunningConfigurationInput struct {
	OrgId            string
	Name             string
	Description      string
	Priority         int
	AppliesTo        domain.DunningConfigScope
	TargetRules      map[string]any
	Config           domain.DunningConfig
	IsAbTest         bool
	AbTestPercentage float64
	CreatedBy        string
}

// UpdateDunningConfigurationInput is the input for DunningService.UpdateConfiguration.
type UpdateDunningConfigurationInput struct {
	OrgId            string
	Id               string
	Name             string
	Description      string
	Priority         int
	AppliesTo        domain.DunningConfigScope
	TargetRules      map[string]any
	Config           *domain.DunningConfig
	Status           domain.ConfigStatus
	IsAbTest         *bool
	AbTestPercentage *float64
}

// CreatePaymentUpdateTokenInput is the input for DunningService.CreatePaymentUpdateToken.
type CreatePaymentUpdateTokenInput struct {
	OrgId             string
	SubscriptionId    string
	CustomerId        string
	DunningCampaignId string
	MaxUses           int
	ExpiryHours       int
	AllowedActions    map[string]bool
	AdminGenerated    bool
	AdminUserId       string
	AdminReason       string
	AdminNotes        string
	CreatedBy         string
}

// ActivatePaymentUpdateTokenInput is the input for DunningService.ActivatePaymentUpdateToken.
type ActivatePaymentUpdateTokenInput struct {
	OrgId   string
	TokenId string
	UsedIp  string
}
