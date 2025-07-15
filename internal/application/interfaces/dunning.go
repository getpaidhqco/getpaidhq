package interfaces

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
)

// DunningService defines the interface for dunning service operations
type DunningService interface {
	// Campaign operations
	CreateCampaign(ctx context.Context, input dto.CreateDunningCampaignInput) (dunning.DunningCampaign, error)
	FindCampaignById(ctx context.Context, orgId string, id string) (dunning.DunningCampaign, error)
	ListCampaigns(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error)
	ListCampaignsBySubscription(ctx context.Context, orgId string, subscriptionId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error)
	ListCampaignsByCustomer(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error)
	PauseCampaign(ctx context.Context, input dto.PauseDunningCampaignInput) (dunning.DunningCampaign, error)
	ResumeCampaign(ctx context.Context, input dto.ResumeDunningCampaignInput) (dunning.DunningCampaign, error)
	CancelCampaign(ctx context.Context, input dto.CancelDunningCampaignInput) (dunning.DunningCampaign, error)
	UpdateCampaign(ctx context.Context, orgId string, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error)

	// Attempt operations
	ListAttemptsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningAttempt, int, error)
	TriggerChargeAttempt(ctx context.Context, input dto.TriggerAttemptInput) (dunning.DunningAttempt, error)
	HandleChargeResult(ctx context.Context, campaign dunning.DunningCampaign, chargeResult payments.ChargeResult, config dunning.DunningConfig) (dto.HandleChargeResultResponse, error)
	// Communication operations
	ListCommunicationsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningCommunication, int, error)

	// Token operations
	CreatePaymentUpdateToken(ctx context.Context, input dto.CreatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error)
	VerifyPaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error)
	ActivatePaymentUpdateToken(ctx context.Context, input dto.ActivatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error)
	RevokePaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error)

	// Configuration operations
	CreateConfiguration(ctx context.Context, input dto.CreateDunningConfigurationInput) (dunning.DunningConfiguration, error)
	GetConfiguration(ctx context.Context, orgId string, id string) (dunning.DunningConfiguration, error)
	ListConfigurations(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningConfiguration, int, error)
	UpdateConfiguration(ctx context.Context, input dto.UpdateDunningConfigurationInput) (dunning.DunningConfiguration, error)

	// Customer dunning history operations
	GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (dunning.CustomerDunningHistory, error)
}

// DunningOrchestrationService defines the interface for dunning workflow operations
type DunningOrchestrationService interface {
	DunningService

	// Workflow operations
	StartDunningWorkflow(ctx context.Context, input dto.StartDunningWorkflowInput) (dunning.DunningCampaign, error)
	HandlePaymentMethodUpdated(ctx context.Context, input dto.PaymentMethodUpdatedInput) error
	HandleSubscriptionStateChanged(ctx context.Context, input dto.SubscriptionStateChangedInput) error
	HandleDunningAttemptResult(ctx context.Context, input dto.DunningAttemptResultInput) (dunning.DunningCampaign, error)
}
