package repositories

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
)

// DunningRepository defines the interface for dunning campaign operations
type DunningRepository interface {
	// Campaign operations
	CreateCampaign(ctx context.Context, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error)
	FindCampaignById(ctx context.Context, orgId string, id string) (dunning.DunningCampaign, error)
	FindCampaigns(ctx context.Context, orgId string, pagination entities.Pagination) ([]dunning.DunningCampaign, int, error)
	FindCampaignsBySubscriptionId(ctx context.Context, orgId string, subscriptionId string, pagination entities.Pagination) ([]dunning.DunningCampaign, int, error)
	FindCampaignsByCustomerId(ctx context.Context, orgId string, customerId string, pagination entities.Pagination) ([]dunning.DunningCampaign, int, error)
	UpdateCampaign(ctx context.Context, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error)
	
	// Attempt operations
	CreateAttempt(ctx context.Context, attempt dunning.DunningAttempt) (dunning.DunningAttempt, error)
	FindAttemptById(ctx context.Context, orgId string, id string) (dunning.DunningAttempt, error)
	FindAttemptsByCampaignId(ctx context.Context, orgId string, campaignId string, pagination entities.Pagination) ([]dunning.DunningAttempt, int, error)
	
	// Communication operations
	CreateCommunication(ctx context.Context, communication dunning.DunningCommunication) (dunning.DunningCommunication, error)
	FindCommunicationById(ctx context.Context, orgId string, id string) (dunning.DunningCommunication, error)
	FindCommunicationsByCampaignId(ctx context.Context, orgId string, campaignId string, pagination entities.Pagination) ([]dunning.DunningCommunication, int, error)
	UpdateCommunication(ctx context.Context, communication dunning.DunningCommunication) (dunning.DunningCommunication, error)
	
	// Token operations
	CreateToken(ctx context.Context, token dunning.PaymentUpdateToken) (dunning.PaymentUpdateToken, error)
	FindTokenById(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error)
	FindTokensBySubscriptionId(ctx context.Context, orgId string, subscriptionId string, pagination entities.Pagination) ([]dunning.PaymentUpdateToken, int, error)
	FindTokensByCampaignId(ctx context.Context, orgId string, campaignId string, pagination entities.Pagination) ([]dunning.PaymentUpdateToken, int, error)
	UpdateToken(ctx context.Context, token dunning.PaymentUpdateToken) (dunning.PaymentUpdateToken, error)
	
	// Configuration operations
	CreateConfiguration(ctx context.Context, config dunning.DunningConfiguration) (dunning.DunningConfiguration, error)
	FindConfigurationById(ctx context.Context, orgId string, id string) (dunning.DunningConfiguration, error)
	FindConfigurations(ctx context.Context, orgId string, pagination entities.Pagination) ([]dunning.DunningConfiguration, int, error)
	UpdateConfiguration(ctx context.Context, config dunning.DunningConfiguration) (dunning.DunningConfiguration, error)
	
	// Customer dunning history operations
	GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (dunning.CustomerDunningHistory, error)
	UpdateCustomerDunningHistory(ctx context.Context, history dunning.CustomerDunningHistory) (dunning.CustomerDunningHistory, error)
}