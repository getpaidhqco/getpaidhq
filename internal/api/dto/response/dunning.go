package response

import (
	"payloop/internal/domain/entities/dunning"
	"time"
)

// DunningCampaignResponse represents a dunning campaign response
type DunningCampaignResponse struct {
	ID                   string            `json:"id"`
	SubscriptionID       string            `json:"subscription_id"`
	CustomerID           string            `json:"customer_id"`
	Status               string            `json:"status"`
	FailedAmount         int               `json:"failed_amount"`
	Currency             string            `json:"currency"`
	InitialFailureReason string            `json:"initial_failure_reason,omitempty"`
	TotalAttempts        int               `json:"total_attempts"`
	ImmediateAttempts    int               `json:"immediate_attempts"`
	ProgressiveAttempts  int               `json:"progressive_attempts"`
	StartedAt            time.Time         `json:"started_at"`
	LastAttemptAt        time.Time         `json:"last_attempt_at,omitempty"`
	NextAttemptAt        time.Time         `json:"next_attempt_at,omitempty"`
	CompletedAt          time.Time         `json:"completed_at,omitempty"`
	RecoveryMethod       string            `json:"recovery_method,omitempty"`
	RecoveredAmount      int               `json:"recovered_amount,omitempty"`
	RecoveredAt          time.Time         `json:"recovered_at,omitempty"`
	FinalFailureReason   string            `json:"final_failure_reason,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// DunningAttemptResponse represents a dunning attempt response
type DunningAttemptResponse struct {
	ID                string            `json:"id"`
	DunningCampaignID string            `json:"dunning_campaign_id"`
	SubscriptionID    string            `json:"subscription_id"`
	AttemptNumber     int               `json:"attempt_number"`
	AttemptType       string            `json:"attempt_type"`
	Amount            int64             `json:"amount"`
	Currency          string            `json:"currency"`
	PaymentMethodID   string            `json:"payment_method_id,omitempty"`
	Status            string            `json:"status"`
	FailureReason     string            `json:"failure_reason,omitempty"`
	FailureCode       string            `json:"failure_code,omitempty"`
	ProcessingTimeMs  int               `json:"processing_time_ms,omitempty"`
	AttemptedAt       time.Time         `json:"attempted_at"`
	CompletedAt       time.Time         `json:"completed_at,omitempty"`
	TriggeredBy       string            `json:"triggered_by,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
}

// DunningCommunicationResponse represents a dunning communication response
type DunningCommunicationResponse struct {
	ID                string    `json:"id"`
	DunningCampaignID string    `json:"dunning_campaign_id"`
	CustomerID        string    `json:"customer_id"`
	Channel           string    `json:"channel"`
	TemplateID        string    `json:"template_id"`
	AttemptNumber     int       `json:"attempt_number"`
	Subject           string    `json:"subject,omitempty"`
	ContentPreview    string    `json:"content_preview,omitempty"`
	SentAt            time.Time `json:"sent_at,omitempty"`
	DeliveredAt       time.Time `json:"delivered_at,omitempty"`
	OpenedAt          time.Time `json:"opened_at,omitempty"`
	ClickedAt         time.Time `json:"clicked_at,omitempty"`
	BouncedAt         time.Time `json:"bounced_at,omitempty"`
	Provider          string    `json:"provider"`
	Status            string    `json:"status"`
	FailureReason     string    `json:"failure_reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// PaymentUpdateTokenResponse represents a payment update token response
type PaymentUpdateTokenResponse struct {
	TokenID           string          `json:"token_id"`
	SubscriptionID    string          `json:"subscription_id"`
	CustomerID        string          `json:"customer_id"`
	DunningCampaignID string          `json:"dunning_campaign_id,omitempty"`
	ExpiresAt         time.Time       `json:"expires_at"`
	MaxUses           int             `json:"max_uses"`
	UsedCount         int             `json:"used_count"`
	Status            string          `json:"status"`
	AllowedActions    map[string]bool `json:"allowed_actions"`
	AdminGenerated    bool            `json:"admin_generated"`
	CreatedBy         string          `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
	LastUsedAt        time.Time       `json:"last_used_at,omitempty"`
}

// DunningConfigurationResponse represents a dunning configuration response
type DunningConfigurationResponse struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Description      string                     `json:"description,omitempty"`
	Priority         int                        `json:"priority"`
	AppliesTo        dunning.DunningConfigScope `json:"applies_to"`
	TargetRules      map[string]interface{}     `json:"target_rules,omitempty"`
	Config           map[string]interface{}     `json:"config"`
	Status           string                     `json:"status"`
	IsAbTest         bool                       `json:"is_ab_test"`
	AbTestPercentage float64                    `json:"ab_test_percentage,omitempty"`
	CreatedBy        string                     `json:"created_by,omitempty"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

// CustomerDunningHistoryResponse represents a customer dunning history response
type CustomerDunningHistoryResponse struct {
	CustomerID              string                       `json:"customer_id"`
	TotalDunningCampaigns   int                          `json:"total_dunning_campaigns"`
	SuccessfulRecoveries    int                          `json:"successful_recoveries"`
	FailedCampaigns         int                          `json:"failed_campaigns"`
	TotalAmountAtRisk       int                          `json:"total_amount_at_risk"`
	TotalAmountRecovered    int                          `json:"total_amount_recovered"`
	TotalAmountLost         int                          `json:"total_amount_lost"`
	AvgRecoveryTimeHours    float64                      `json:"avg_recovery_time_hours,omitempty"`
	PreferredRecoveryMethod string                       `json:"preferred_recovery_method,omitempty"`
	MostResponsiveChannel   dunning.CommunicationChannel `json:"most_responsive_channel,omitempty"`
	PaymentReliabilityScore float64                      `json:"payment_reliability_score,omitempty"`
	DunningRiskTier         string                       `json:"dunning_risk_tier,omitempty"`
	FirstDunningAt          time.Time                    `json:"first_dunning_at,omitempty"`
	LastDunningAt           time.Time                    `json:"last_dunning_at,omitempty"`
	LastRecoveryAt          time.Time                    `json:"last_recovery_at,omitempty"`
}

// FromDunningCampaign converts a dunning.DunningCampaign to a DunningCampaignResponse
func FromDunningCampaign(campaign dunning.DunningCampaign) DunningCampaignResponse {
	return DunningCampaignResponse{
		ID:                   campaign.Id,
		SubscriptionID:       campaign.SubscriptionId,
		CustomerID:           campaign.CustomerId,
		Status:               string(campaign.Status),
		FailedAmount:         campaign.FailedAmount,
		Currency:             campaign.Currency,
		InitialFailureReason: campaign.InitialFailureReason,
		TotalAttempts:        campaign.TotalAttempts,
		ImmediateAttempts:    campaign.ImmediateAttempts,
		ProgressiveAttempts:  campaign.ProgressiveAttempts,
		StartedAt:            campaign.StartedAt,
		LastAttemptAt:        campaign.LastAttemptAt,
		NextAttemptAt:        campaign.NextAttemptAt,
		CompletedAt:          campaign.CompletedAt,
		RecoveryMethod:       campaign.RecoveryMethod,
		RecoveredAmount:      campaign.RecoveredAmount,
		RecoveredAt:          campaign.RecoveredAt,
		FinalFailureReason:   campaign.FinalFailureReason,
		Metadata:             campaign.Metadata,
		CreatedAt:            campaign.CreatedAt,
		UpdatedAt:            campaign.UpdatedAt,
	}
}

// FromDunningAttempt converts a dunning.DunningAttempt to a DunningAttemptResponse
func FromDunningAttempt(attempt dunning.DunningAttempt) DunningAttemptResponse {
	return DunningAttemptResponse{
		ID:                attempt.Id,
		DunningCampaignID: attempt.DunningCampaignId,
		SubscriptionID:    attempt.SubscriptionId,
		AttemptNumber:     attempt.AttemptNumber,
		AttemptType:       string(attempt.AttemptType),
		Amount:            attempt.Amount,
		Currency:          attempt.Currency,
		PaymentMethodID:   attempt.PaymentMethodId,
		Status:            string(attempt.Status),
		FailureReason:     attempt.FailureReason,
		FailureCode:       attempt.FailureCode,
		ProcessingTimeMs:  attempt.ProcessingTimeMs,
		AttemptedAt:       attempt.AttemptedAt,
		CompletedAt:       attempt.CompletedAt,
		TriggeredBy:       attempt.TriggeredBy,
		Metadata:          attempt.Metadata,
		CreatedAt:         attempt.CreatedAt,
	}
}

// FromDunningCommunication converts a dunning.DunningCommunication to a DunningCommunicationResponse
func FromDunningCommunication(communication dunning.DunningCommunication) DunningCommunicationResponse {
	return DunningCommunicationResponse{
		ID:                communication.Id,
		DunningCampaignID: communication.DunningCampaignId,
		CustomerID:        communication.CustomerId,
		Channel:           string(communication.Channel),
		TemplateID:        communication.TemplateId,
		AttemptNumber:     communication.AttemptNumber,
		Subject:           communication.Subject,
		ContentPreview:    communication.ContentPreview,
		SentAt:            communication.SentAt,
		DeliveredAt:       communication.DeliveredAt,
		OpenedAt:          communication.OpenedAt,
		ClickedAt:         communication.ClickedAt,
		BouncedAt:         communication.BouncedAt,
		Provider:          communication.Provider,
		Status:            string(communication.Status),
		FailureReason:     communication.FailureReason,
		CreatedAt:         communication.CreatedAt,
		UpdatedAt:         communication.UpdatedAt,
	}
}

// FromPaymentUpdateToken converts a dunning.PaymentUpdateToken to a PaymentUpdateTokenResponse
func FromPaymentUpdateToken(token dunning.PaymentUpdateToken) PaymentUpdateTokenResponse {
	return PaymentUpdateTokenResponse{
		TokenID:           token.TokenId,
		SubscriptionID:    token.SubscriptionId,
		CustomerID:        token.CustomerId,
		DunningCampaignID: token.DunningCampaignId,
		ExpiresAt:         token.ExpiresAt,
		MaxUses:           token.MaxUses,
		UsedCount:         token.UsedCount,
		Status:            string(token.Status),
		AllowedActions:    token.AllowedActions,
		AdminGenerated:    token.AdminGenerated,
		CreatedBy:         token.CreatedBy,
		CreatedAt:         token.CreatedAt,
		LastUsedAt:        token.LastUsedAt,
	}
}

// FromDunningConfiguration converts a dunning.DunningConfiguration to a DunningConfigurationResponse
func FromDunningConfiguration(config dunning.DunningConfiguration) DunningConfigurationResponse {
	return DunningConfigurationResponse{
		ID:               config.Id,
		Name:             config.Name,
		Description:      config.Description,
		Priority:         config.Priority,
		AppliesTo:        config.AppliesTo,
		TargetRules:      config.TargetRules,
		Config:           config.Config,
		Status:           string(config.Status),
		IsAbTest:         config.IsAbTest,
		AbTestPercentage: config.AbTestPercentage,
		CreatedBy:        config.CreatedBy,
		CreatedAt:        config.CreatedAt,
		UpdatedAt:        config.UpdatedAt,
	}
}

// FromCustomerDunningHistory converts a dunning.CustomerDunningHistory to a CustomerDunningHistoryResponse
func FromCustomerDunningHistory(history dunning.CustomerDunningHistory) CustomerDunningHistoryResponse {
	return CustomerDunningHistoryResponse{
		CustomerID:              history.CustomerId,
		TotalDunningCampaigns:   history.TotalDunningCampaigns,
		SuccessfulRecoveries:    history.SuccessfulRecoveries,
		FailedCampaigns:         history.FailedCampaigns,
		TotalAmountAtRisk:       history.TotalAmountAtRisk,
		TotalAmountRecovered:    history.TotalAmountRecovered,
		TotalAmountLost:         history.TotalAmountLost,
		AvgRecoveryTimeHours:    history.AvgRecoveryTimeHours,
		PreferredRecoveryMethod: history.PreferredRecoveryMethod,
		MostResponsiveChannel:   history.MostResponsiveChannel,
		PaymentReliabilityScore: history.PaymentReliabilityScore,
		DunningRiskTier:         history.DunningRiskTier,
		FirstDunningAt:          history.FirstDunningAt,
		LastDunningAt:           history.LastDunningAt,
		LastRecoveryAt:          history.LastRecoveryAt,
	}
}
