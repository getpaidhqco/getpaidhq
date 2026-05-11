package domain

// CreateDunningCampaignInput is what the orchestrator passes to
// DunningService.CreateCampaign when a charge fails.
type CreateDunningCampaignInput struct {
	OrgId                string
	SubscriptionId       string
	CustomerId           string
	FailedAmount         int64
	Currency             string
	InitialFailureReason string
	ParentWorkflowId     string
	Metadata             map[string]string
}

// StartDunningWorkflowInput is the payload the orchestrator hands to
// port.Engine.StartDunningWorkflow.
type StartDunningWorkflowInput struct {
	OrgId                string
	SubscriptionId       string
	CustomerId           string
	FailedAmount         int64
	Currency             string
	InitialFailureReason string
	ParentWorkflowId     string
	PaymentResult        ChargeResult
	Metadata             map[string]string
}

// PauseDunningCampaignInput / ResumeDunningCampaignInput / CancelDunningCampaignInput
// carry a reason that ends up on the campaign / its outbound events.
type PauseDunningCampaignInput struct {
	OrgId      string
	CampaignId string
	Reason     string
}

type ResumeDunningCampaignInput struct {
	OrgId      string
	CampaignId string
	Reason     string
}

type CancelDunningCampaignInput struct {
	OrgId      string
	CampaignId string
	Reason     string
}

type TriggerManualAttemptInput struct {
	OrgId           string
	CampaignId      string
	PaymentMethodId string
	TriggeredBy     string
}

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

type ActivatePaymentUpdateTokenInput struct {
	OrgId   string
	TokenId string
	UsedIp  string
}

type CreateDunningConfigurationInput struct {
	OrgId            string
	Name             string
	Description      string
	Priority         int
	AppliesTo        DunningConfigScope
	TargetRules      map[string]any
	Config           DunningConfig
	IsAbTest         bool
	AbTestPercentage float64
	CreatedBy        string
}

type UpdateDunningConfigurationInput struct {
	OrgId            string
	Id               string
	Name             string
	Description      string
	Priority         int
	AppliesTo        DunningConfigScope
	TargetRules      map[string]any
	Config           *DunningConfig
	Status           ConfigStatus
	IsAbTest         *bool
	AbTestPercentage *float64
}

type SubscriptionStateChangedInput struct {
	OrgId          string
	CampaignId     string
	SubscriptionId string
	OldStatus      SubscriptionStatus
	NewStatus      SubscriptionStatus
}

// DunningAttemptContext is passed to UpdateCampaignWithAttemptResult so the
// engine adapter doesn't need to re-derive escalation state.
type DunningAttemptContext struct {
	AttemptNumber            int
	WasSubscriptionSuspended bool
}
