package port

import (
	"context"
	"payloop/internal/core/domain"
)

// DunningRepository owns persistence for the dunning aggregate (campaigns,
// attempts, communications, configurations, tokens, customer history).
type DunningRepository interface {
	// Campaigns
	CreateCampaign(ctx context.Context, campaign domain.DunningCampaign) (domain.DunningCampaign, error)
	FindCampaignById(ctx context.Context, orgId string, id string) (domain.DunningCampaign, error)
	FindCampaigns(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningCampaign, int, error)
	FindCampaignsBySubscriptionId(ctx context.Context, orgId string, subscriptionId string, p domain.Pagination) ([]domain.DunningCampaign, int, error)
	FindCampaignsByCustomerId(ctx context.Context, orgId string, customerId string, p domain.Pagination) ([]domain.DunningCampaign, int, error)
	FindActiveCampaignForSubscription(ctx context.Context, orgId string, subscriptionId string) (domain.DunningCampaign, error)
	UpdateCampaign(ctx context.Context, campaign domain.DunningCampaign) (domain.DunningCampaign, error)

	// Attempts
	CreateAttempt(ctx context.Context, attempt domain.DunningAttempt) (domain.DunningAttempt, error)
	FindAttemptById(ctx context.Context, orgId string, id string) (domain.DunningAttempt, error)
	FindAttemptsByCampaignId(ctx context.Context, orgId string, campaignId string, p domain.Pagination) ([]domain.DunningAttempt, int, error)

	// Communications
	CreateCommunication(ctx context.Context, communication domain.DunningCommunication) (domain.DunningCommunication, error)
	FindCommunicationById(ctx context.Context, orgId string, id string) (domain.DunningCommunication, error)
	FindCommunicationsByCampaignId(ctx context.Context, orgId string, campaignId string, p domain.Pagination) ([]domain.DunningCommunication, int, error)
	UpdateCommunication(ctx context.Context, communication domain.DunningCommunication) (domain.DunningCommunication, error)

	// Tokens
	CreateToken(ctx context.Context, token domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error)
	FindTokenById(ctx context.Context, orgId string, tokenId string) (domain.PaymentUpdateToken, error)
	FindTokensBySubscriptionId(ctx context.Context, orgId string, subscriptionId string, p domain.Pagination) ([]domain.PaymentUpdateToken, int, error)
	FindTokensByCampaignId(ctx context.Context, orgId string, campaignId string, p domain.Pagination) ([]domain.PaymentUpdateToken, int, error)
	UpdateToken(ctx context.Context, token domain.PaymentUpdateToken) (domain.PaymentUpdateToken, error)

	// Configurations
	CreateConfiguration(ctx context.Context, config domain.DunningConfiguration) (domain.DunningConfiguration, error)
	FindConfigurationById(ctx context.Context, orgId string, id string) (domain.DunningConfiguration, error)
	FindConfigurations(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningConfiguration, int, error)
	FindConfigurationsByPriority(ctx context.Context, orgId string) ([]domain.DunningConfiguration, error)
	UpdateConfiguration(ctx context.Context, config domain.DunningConfiguration) (domain.DunningConfiguration, error)

	// Customer history
	GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (domain.CustomerDunningHistory, error)
	UpsertCustomerDunningHistory(ctx context.Context, history domain.CustomerDunningHistory) (domain.CustomerDunningHistory, error)
}

// DunningService is the engine-agnostic dunning aggregate root. Activities/
// steps depend on this; HTTP handlers depend on DunningOrchestrationService.
type DunningService interface {
	// Campaign lifecycle
	CreateCampaign(ctx context.Context, input domain.CreateDunningCampaignInput) (domain.DunningCampaign, error)
	FindCampaignById(ctx context.Context, orgId string, id string) (domain.DunningCampaign, error)
	ListCampaigns(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningCampaign, int, error)
	ListCampaignsBySubscription(ctx context.Context, orgId string, subscriptionId string, p domain.Pagination) ([]domain.DunningCampaign, int, error)
	ListCampaignsByCustomer(ctx context.Context, orgId string, customerId string, p domain.Pagination) ([]domain.DunningCampaign, int, error)
	PauseCampaign(ctx context.Context, input domain.PauseDunningCampaignInput) (domain.DunningCampaign, error)
	ResumeCampaign(ctx context.Context, input domain.ResumeDunningCampaignInput) (domain.DunningCampaign, error)
	CancelCampaign(ctx context.Context, input domain.CancelDunningCampaignInput) (domain.DunningCampaign, error)
	UpdateCampaign(ctx context.Context, campaign domain.DunningCampaign) (domain.DunningCampaign, error)
	MarkCampaignRecovered(ctx context.Context, orgId string, campaignId string, recoveryMethod string, recoveredAmount int64) (domain.DunningCampaign, error)
	MarkCampaignFailed(ctx context.Context, orgId string, campaignId string, finalFailureReason string) (domain.DunningCampaign, error)
	// FailCampaignAndCancelSubscription marks the campaign Failed and cancels
	// the underlying subscription in one shot. Used by the runner when both
	// retry phases exhaust without crossing an explicit escalation threshold.
	FailCampaignAndCancelSubscription(ctx context.Context, orgId string, campaignId string, finalFailureReason string) (domain.DunningCampaign, error)

	// Attempts
	ListAttemptsByCampaign(ctx context.Context, orgId string, campaignId string, p domain.Pagination) ([]domain.DunningAttempt, int, error)
	TriggerManualAttempt(ctx context.Context, input domain.TriggerManualAttemptInput) (domain.DunningAttempt, error)
	UpdateCampaignWithAttemptResult(ctx context.Context, attempt domain.DunningAttempt, config domain.DunningConfig, attemptContext domain.DunningAttemptContext) (domain.DunningCampaign, error)
	ExecuteAttempt(ctx context.Context, orgId string, campaignId string, attemptType domain.DunningAttemptType) (domain.DunningAttempt, error)

	// Communications
	ListCommunicationsByCampaign(ctx context.Context, orgId string, campaignId string, p domain.Pagination) ([]domain.DunningCommunication, int, error)
	SendCommunication(ctx context.Context, orgId string, campaignId string, attemptNumber int) error

	// Tokens
	CreatePaymentUpdateToken(ctx context.Context, input domain.CreatePaymentUpdateTokenInput) (domain.PaymentUpdateToken, error)
	VerifyPaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (domain.PaymentUpdateToken, error)
	ActivatePaymentUpdateToken(ctx context.Context, input domain.ActivatePaymentUpdateTokenInput) (domain.PaymentUpdateToken, error)
	RevokePaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (domain.PaymentUpdateToken, error)

	// Configurations
	CreateConfiguration(ctx context.Context, input domain.CreateDunningConfigurationInput) (domain.DunningConfiguration, error)
	GetConfiguration(ctx context.Context, orgId string, id string) (domain.DunningConfiguration, error)
	ListConfigurations(ctx context.Context, orgId string, p domain.Pagination) ([]domain.DunningConfiguration, int, error)
	UpdateConfiguration(ctx context.Context, input domain.UpdateDunningConfigurationInput) (domain.DunningConfiguration, error)
	ResolveConfig(ctx context.Context, orgId string) (domain.DunningConfig, error)
	// LoadConfigForCampaign prefers the snapshot stored on the campaign at
	// start time; falls back to ResolveConfig if the snapshot is empty (e.g.
	// for in-flight campaigns started before the snapshot field was populated).
	LoadConfigForCampaign(ctx context.Context, orgId string, campaignId string) (domain.DunningConfig, error)

	// Customer history
	GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (domain.CustomerDunningHistory, error)
}

// DunningEngine is the dunning-specific addition to the workflow engine
// surface. Kept separate from port.Engine so the existing surface stays
// engine-agnostic and the dunning surface can be implemented adapter-by-adapter.
//
// Returns (workflowId, workflowRunId, error). The workflow handle is stored on
// the campaign so the orchestrator can address it later (signals, cancel).
type DunningEngine interface {
	StartDunningWorkflow(ctx context.Context, input domain.StartDunningWorkflowInput) (string, string, error)
	SignalDunningWorkflow(ctx context.Context, signal string, campaign domain.DunningCampaign, payload any) error
	CancelDunningWorkflow(ctx context.Context, campaign domain.DunningCampaign) error
}
