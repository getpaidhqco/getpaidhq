package port

import (
	"payloop/internal/core/domain"
)

// DunningCampaignEvent is published on campaign lifecycle transitions
// (started, paused, resumed, cancelled, recovered, failed, expired).
type DunningCampaignEvent struct {
	OrgId               string               `json:"org_id"`
	CampaignId          string               `json:"campaign_id"`
	SubscriptionId      string               `json:"subscription_id"`
	CustomerId          string               `json:"customer_id"`
	Status              domain.DunningStatus `json:"status"`
	FailedAmount        int64                `json:"failed_amount"`
	TotalAttempts       int                  `json:"total_attempts"`
	ImmediateAttempts   int                  `json:"immediate_attempts"`
	ProgressiveAttempts int                  `json:"progressive_attempts"`
	Currency            string               `json:"currency"`
	Metadata            map[string]string    `json:"metadata"`
}

func NewDunningCampaignEvent(c domain.DunningCampaign) DunningCampaignEvent {
	return DunningCampaignEvent{
		OrgId:               c.OrgId,
		CampaignId:          c.Id,
		SubscriptionId:      c.SubscriptionId,
		CustomerId:          c.CustomerId,
		Status:              c.Status,
		FailedAmount:        c.FailedAmount,
		TotalAttempts:       c.TotalAttempts,
		ImmediateAttempts:   c.ImmediateAttempts,
		ProgressiveAttempts: c.ProgressiveAttempts,
		Currency:            c.Currency,
		Metadata:            c.Metadata,
	}
}

// DunningAttemptEvent is published on every attempt outcome.
type DunningAttemptEvent struct {
	OrgId          string                    `json:"org_id"`
	CampaignId     string                    `json:"campaign_id"`
	AttemptId      string                    `json:"attempt_id"`
	SubscriptionId string                    `json:"subscription_id"`
	CustomerId     string                    `json:"customer_id"`
	AttemptNumber  int                       `json:"attempt_number"`
	AttemptType    domain.DunningAttemptType `json:"attempt_type"`
	Amount         int64                     `json:"amount"`
	Currency       string                    `json:"currency"`
	Status         domain.PaymentStatus      `json:"status"`
	FailureReason  string                    `json:"failure_reason"`
	FailureCode    string                    `json:"failure_code"`
	ShouldSuspend  bool                      `json:"should_suspend"`
	IsFinalNotice  bool                      `json:"is_final_notice"`
	Metadata       map[string]string         `json:"metadata"`
}

func NewDunningAttemptEvent(a domain.DunningAttempt, customerId string, shouldSuspend, isFinalNotice bool) DunningAttemptEvent {
	return DunningAttemptEvent{
		OrgId:          a.OrgId,
		CampaignId:     a.DunningCampaignId,
		AttemptId:      a.Id,
		SubscriptionId: a.SubscriptionId,
		CustomerId:     customerId,
		AttemptNumber:  a.AttemptNumber,
		AttemptType:    a.AttemptType,
		Amount:         a.Amount,
		Currency:       a.Currency,
		Status:         a.Status,
		FailureReason:  a.FailureReason,
		FailureCode:    a.FailureCode,
		ShouldSuspend:  shouldSuspend,
		IsFinalNotice:  isFinalNotice,
		Metadata:       a.Metadata,
	}
}

// DunningCommunicationEvent is published when a dunning communication is
// dispatched (or fails to dispatch) downstream notification consumers.
type DunningCommunicationEvent struct {
	OrgId           string                      `json:"org_id"`
	CampaignId      string                      `json:"campaign_id"`
	CommunicationId string                      `json:"communication_id"`
	CustomerId      string                      `json:"customer_id"`
	Channel         domain.CommunicationChannel `json:"channel"`
	TemplateId      string                      `json:"template_id"`
	AttemptNumber   int                         `json:"attempt_number"`
	Status          domain.CommunicationStatus  `json:"status"`
	FailureReason   string                      `json:"failure_reason"`
}

// DunningTokenEvent is published on token lifecycle transitions.
type DunningTokenEvent struct {
	OrgId          string             `json:"org_id"`
	TokenId        string             `json:"token_id"`
	SubscriptionId string             `json:"subscription_id"`
	CustomerId     string             `json:"customer_id"`
	CampaignId     string             `json:"campaign_id"`
	Status         domain.TokenStatus `json:"status"`
	ExpiresAt      string             `json:"expires_at"`
	MaxUses        int                `json:"max_uses"`
	UsedCount      int                `json:"used_count"`
	AllowedActions map[string]bool    `json:"allowed_actions"`
	AdminGenerated bool               `json:"admin_generated"`
}

func NewDunningTokenEvent(t domain.PaymentUpdateToken) DunningTokenEvent {
	return DunningTokenEvent{
		OrgId:          t.OrgId,
		TokenId:        t.TokenId,
		SubscriptionId: t.SubscriptionId,
		CustomerId:     t.CustomerId,
		CampaignId:     t.DunningCampaignId,
		Status:         t.Status,
		ExpiresAt:      t.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		MaxUses:        t.MaxUses,
		UsedCount:      t.UsedCount,
		AllowedActions: t.AllowedActions,
		AdminGenerated: t.AdminGenerated,
	}
}

// DunningConfigurationEvent is published on configuration mutations.
type DunningConfigurationEvent struct {
	OrgId     string                    `json:"org_id"`
	ConfigId  string                    `json:"config_id"`
	Name      string                    `json:"name"`
	AppliesTo domain.DunningConfigScope `json:"applies_to"`
	Status    domain.ConfigStatus       `json:"status"`
}

func NewDunningConfigurationEvent(c domain.DunningConfiguration) DunningConfigurationEvent {
	return DunningConfigurationEvent{
		OrgId:     c.OrgId,
		ConfigId:  c.Id,
		Name:      c.Name,
		AppliesTo: c.AppliesTo,
		Status:    c.Status,
	}
}

// DunningSubscriptionEvent is used when dunning forces a subscription state
// transition (suspend / reactivate) so downstream consumers can react.
type DunningSubscriptionEvent struct {
	OrgId          string                    `json:"org_id"`
	CampaignId     string                    `json:"campaign_id"`
	SubscriptionId string                    `json:"subscription_id"`
	CustomerId     string                    `json:"customer_id"`
	OldStatus      domain.SubscriptionStatus `json:"old_status"`
	NewStatus      domain.SubscriptionStatus `json:"new_status"`
}
