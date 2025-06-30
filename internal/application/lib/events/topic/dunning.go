package topic

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"time"
)

// DunningCampaignEvent is the base event for dunning campaign events
type DunningCampaignEvent struct {
	OrgId             string                 `json:"org_id"`
	CampaignId        string                 `json:"campaign_id"`
	SubscriptionId    string                 `json:"subscription_id"`
	CustomerId        string                 `json:"customer_id"`
	Status            dunning.DunningStatus  `json:"status"`
	FailedAmount      int                    `json:"failed_amount"`
	Currency          string                 `json:"currency"`
	TotalAttempts     int                    `json:"total_attempts"`
	ImmediateAttempts int                    `json:"immediate_attempts"`
	ProgressiveAttempts int                  `json:"progressive_attempts"`
	Metadata          map[string]string      `json:"metadata,omitempty"`
}

// DunningAttemptEvent represents an event for a dunning attempt
type DunningAttemptEvent struct {
	OrgId             string                 `json:"org_id"`
	CampaignId        string                 `json:"campaign_id"`
	AttemptId         string                 `json:"attempt_id"`
	SubscriptionId    string                 `json:"subscription_id"`
	CustomerId        string                 `json:"customer_id"`
	AttemptNumber     int                    `json:"attempt_number"`
	AttemptType       dunning.DunningAttemptType `json:"attempt_type"`
	Amount            int                    `json:"amount"`
	Currency          string                 `json:"currency"`
	Status            payments.PaymentStatus `json:"status"`
	FailureReason     string                 `json:"failure_reason,omitempty"`
	FailureCode       string                 `json:"failure_code,omitempty"`
	ShouldSuspend     bool                   `json:"should_suspend"`
	IsFinalNotice     bool                   `json:"is_final_notice"`
	Metadata          map[string]string      `json:"metadata,omitempty"`
}

// DunningCommunicationEvent represents an event for a dunning communication
type DunningCommunicationEvent struct {
	OrgId             string                 `json:"org_id"`
	CampaignId        string                 `json:"campaign_id"`
	CommunicationId   string                 `json:"communication_id"`
	CustomerId        string                 `json:"customer_id"`
	Channel           dunning.CommunicationChannel `json:"channel"`
	TemplateId        string                 `json:"template_id"`
	AttemptNumber     int                    `json:"attempt_number"`
	Status            dunning.CommunicationStatus `json:"status"`
	FailureReason     string                 `json:"failure_reason,omitempty"`
}

// DunningTokenEvent represents an event for a payment update token
type DunningTokenEvent struct {
	OrgId             string                 `json:"org_id"`
	TokenId           string                 `json:"token_id"`
	SubscriptionId    string                 `json:"subscription_id"`
	CustomerId        string                 `json:"customer_id"`
	CampaignId        string                 `json:"campaign_id,omitempty"`
	Status            dunning.TokenStatus    `json:"status"`
	ExpiresAt         string                 `json:"expires_at"`
	MaxUses           int                    `json:"max_uses"`
	UsedCount         int                    `json:"used_count"`
	AllowedActions    map[string]bool        `json:"allowed_actions"`
	AdminGenerated    bool                   `json:"admin_generated"`
}

// DunningSubscriptionEvent represents an event for a subscription state change during dunning
type DunningSubscriptionEvent struct {
	OrgId             string                 `json:"org_id"`
	CampaignId        string                 `json:"campaign_id"`
	SubscriptionId    string                 `json:"subscription_id"`
	CustomerId        string                 `json:"customer_id"`
	OldStatus         entities.SubscriptionStatus `json:"old_status"`
	NewStatus         entities.SubscriptionStatus `json:"new_status"`
}

// DunningConfigurationEvent represents an event for a dunning configuration change
type DunningConfigurationEvent struct {
	OrgId             string                 `json:"org_id"`
	ConfigId          string                 `json:"config_id"`
	Name              string                 `json:"name"`
	AppliesTo         dunning.DunningConfigScope `json:"applies_to"`
	Status            dunning.ConfigStatus   `json:"status"`
}

// Helper functions to create events

// NewDunningCampaignEvent creates a new DunningCampaignEvent from a DunningCampaign
func NewDunningCampaignEvent(campaign dunning.DunningCampaign) DunningCampaignEvent {
	return DunningCampaignEvent{
		OrgId:              campaign.OrgId,
		CampaignId:         campaign.Id,
		SubscriptionId:     campaign.SubscriptionId,
		CustomerId:         campaign.CustomerId,
		Status:             campaign.Status,
		FailedAmount:       campaign.FailedAmount,
		Currency:           campaign.Currency,
		TotalAttempts:      campaign.TotalAttempts,
		ImmediateAttempts:  campaign.ImmediateAttempts,
		ProgressiveAttempts: campaign.ProgressiveAttempts,
		Metadata:           campaign.Metadata,
	}
}

// NewDunningAttemptEvent creates a new DunningAttemptEvent from a DunningAttempt
func NewDunningAttemptEvent(attempt dunning.DunningAttempt, campaign dunning.DunningCampaign, shouldSuspend bool, isFinalNotice bool) DunningAttemptEvent {
	return DunningAttemptEvent{
		OrgId:          attempt.OrgId,
		CampaignId:     attempt.DunningCampaignId,
		AttemptId:      attempt.Id,
		SubscriptionId: attempt.SubscriptionId,
		CustomerId:     campaign.CustomerId,
		AttemptNumber:  attempt.AttemptNumber,
		AttemptType:    attempt.AttemptType,
		Amount:         attempt.Amount,
		Currency:       attempt.Currency,
		Status:         attempt.Status,
		FailureReason:  attempt.FailureReason,
		FailureCode:    attempt.FailureCode,
		ShouldSuspend:  shouldSuspend,
		IsFinalNotice:  isFinalNotice,
		Metadata:       attempt.Metadata,
	}
}

// NewDunningCommunicationEvent creates a new DunningCommunicationEvent from a DunningCommunication
func NewDunningCommunicationEvent(communication dunning.DunningCommunication) DunningCommunicationEvent {
	return DunningCommunicationEvent{
		OrgId:           communication.OrgId,
		CampaignId:      communication.DunningCampaignId,
		CommunicationId: communication.Id,
		CustomerId:      communication.CustomerId,
		Channel:         communication.Channel,
		TemplateId:      communication.TemplateId,
		AttemptNumber:   communication.AttemptNumber,
		Status:          communication.Status,
		FailureReason:   communication.FailureReason,
	}
}

// NewDunningTokenEvent creates a new DunningTokenEvent from a PaymentUpdateToken
func NewDunningTokenEvent(token dunning.PaymentUpdateToken) DunningTokenEvent {
	return DunningTokenEvent{
		OrgId:          token.OrgId,
		TokenId:        token.TokenId,
		SubscriptionId: token.SubscriptionId,
		CustomerId:     token.CustomerId,
		CampaignId:     token.DunningCampaignId,
		Status:         token.Status,
		ExpiresAt:      token.ExpiresAt.Format(time.RFC3339),
		MaxUses:        token.MaxUses,
		UsedCount:      token.UsedCount,
		AllowedActions: token.AllowedActions,
		AdminGenerated: token.AdminGenerated,
	}
}

// NewDunningSubscriptionEvent creates a new DunningSubscriptionEvent
func NewDunningSubscriptionEvent(campaign dunning.DunningCampaign, oldStatus, newStatus entities.SubscriptionStatus) DunningSubscriptionEvent {
	return DunningSubscriptionEvent{
		OrgId:          campaign.OrgId,
		CampaignId:     campaign.Id,
		SubscriptionId: campaign.SubscriptionId,
		CustomerId:     campaign.CustomerId,
		OldStatus:      oldStatus,
		NewStatus:      newStatus,
	}
}

// NewDunningConfigurationEvent creates a new DunningConfigurationEvent from a DunningConfiguration
func NewDunningConfigurationEvent(config dunning.DunningConfiguration) DunningConfigurationEvent {
	return DunningConfigurationEvent{
		OrgId:     config.OrgId,
		ConfigId:  config.Id,
		Name:      config.Name,
		AppliesTo: config.AppliesTo,
		Status:    config.Status,
	}
}
