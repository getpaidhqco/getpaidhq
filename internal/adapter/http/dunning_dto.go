package handler

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// ---- requests ----

type UpdateDunningCampaignRequest struct {
	Status string `json:"status" validate:"required,oneof=active paused cancelled"`
	Reason string `json:"reason"`
}

type TriggerManualAttemptRequest struct {
	PaymentMethodID string `json:"payment_method_id"`
}

type VerifyPaymentTokenRequest struct {
	TokenID string `json:"token_id" validate:"required"`
}

type ActivatePaymentTokenRequest struct {
	TokenID string `json:"token_id" validate:"required"`
}

type CreatePaymentTokenRequest struct {
	MaxUses        int             `json:"max_uses"`
	ExpiryHours    int             `json:"expiry_hours"`
	AllowedActions map[string]bool `json:"allowed_actions"`
	AdminReason    string          `json:"admin_reason"`
	AdminNotes     string          `json:"admin_notes"`
}

type CreateDunningConfigurationRequest struct {
	Name             string                    `json:"name" validate:"required"`
	Description      string                    `json:"description"`
	Priority         int                       `json:"priority"`
	AppliesTo        domain.DunningConfigScope `json:"applies_to" validate:"required"`
	TargetRules      map[string]any            `json:"target_rules"`
	Config           domain.DunningConfig      `json:"config" validate:"required"`
	IsAbTest         bool                      `json:"is_ab_test"`
	AbTestPercentage float64                   `json:"ab_test_percentage"`
}

type UpdateDunningConfigurationRequest struct {
	Name             string                    `json:"name"`
	Description      string                    `json:"description"`
	Priority         int                       `json:"priority"`
	AppliesTo        domain.DunningConfigScope `json:"applies_to"`
	TargetRules      map[string]any            `json:"target_rules"`
	Config           *domain.DunningConfig     `json:"config"`
	Status           domain.ConfigStatus       `json:"status"`
	IsAbTest         *bool                     `json:"is_ab_test"`
	AbTestPercentage *float64                  `json:"ab_test_percentage"`
}

// ---- responses ----

type DunningCampaignResponse struct {
	ID                   string            `json:"id"`
	SubscriptionID       string            `json:"subscription_id"`
	CustomerID           string            `json:"customer_id"`
	Status               string            `json:"status"`
	Currency             string            `json:"currency"`
	InitialFailureReason string            `json:"initial_failure_reason,omitempty"`
	FailedAmount         int64             `json:"failed_amount"`
	TotalAttempts        int               `json:"total_attempts"`
	ImmediateAttempts    int               `json:"immediate_attempts"`
	ProgressiveAttempts  int               `json:"progressive_attempts"`
	StartedAt            time.Time         `json:"started_at"`
	LastAttemptAt        time.Time         `json:"last_attempt_at,omitzero"`
	NextAttemptAt        time.Time         `json:"next_attempt_at,omitzero"`
	CompletedAt          time.Time         `json:"completed_at,omitzero"`
	RecoveredAt          time.Time         `json:"recovered_at,omitzero"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	RecoveryMethod       string            `json:"recovery_method,omitempty"`
	FinalFailureReason   string            `json:"final_failure_reason,omitempty"`
	RecoveredAmount      int64             `json:"recovered_amount,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

func NewDunningCampaignResponse(c domain.DunningCampaign) DunningCampaignResponse {
	return DunningCampaignResponse{
		ID:                   c.Id,
		SubscriptionID:       c.SubscriptionId,
		CustomerID:           c.CustomerId,
		Status:               string(c.Status),
		Currency:             c.Currency,
		InitialFailureReason: c.InitialFailureReason,
		FailedAmount:         c.FailedAmount,
		TotalAttempts:        c.TotalAttempts,
		ImmediateAttempts:    c.ImmediateAttempts,
		ProgressiveAttempts:  c.ProgressiveAttempts,
		StartedAt:            timeOrZero(c.StartedAt),
		LastAttemptAt:        timeOrZero(c.LastAttemptAt),
		NextAttemptAt:        timeOrZero(c.NextAttemptAt),
		CompletedAt:          timeOrZero(c.CompletedAt),
		RecoveredAt:          timeOrZero(c.RecoveredAt),
		CreatedAt:            timeOrZero(c.CreatedAt),
		UpdatedAt:            timeOrZero(c.UpdatedAt),
		RecoveryMethod:       c.RecoveryMethod,
		FinalFailureReason:   c.FinalFailureReason,
		RecoveredAmount:      c.RecoveredAmount,
		Metadata:             c.Metadata,
	}
}

type DunningAttemptResponse struct {
	ID                string            `json:"id"`
	DunningCampaignID string            `json:"dunning_campaign_id"`
	SubscriptionID    string            `json:"subscription_id"`
	AttemptType       string            `json:"attempt_type"`
	Currency          string            `json:"currency"`
	Status            string            `json:"status"`
	FailureReason     string            `json:"failure_reason,omitempty"`
	FailureCode       string            `json:"failure_code,omitempty"`
	TriggeredBy       string            `json:"triggered_by,omitempty"`
	AttemptNumber     int               `json:"attempt_number"`
	Amount            int64             `json:"amount"`
	ProcessingTimeMs  int               `json:"processing_time_ms,omitempty"`
	PaymentMethodID   string            `json:"payment_method_id,omitempty"`
	AttemptedAt       time.Time         `json:"attempted_at"`
	CompletedAt       time.Time         `json:"completed_at,omitzero"`
	CreatedAt         time.Time         `json:"created_at"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

func NewDunningAttemptResponse(a domain.DunningAttempt) DunningAttemptResponse {
	return DunningAttemptResponse{
		ID:                a.Id,
		DunningCampaignID: a.DunningCampaignId,
		SubscriptionID:    a.SubscriptionId,
		AttemptType:       string(a.AttemptType),
		Currency:          a.Currency,
		Status:            string(a.Status),
		FailureReason:     a.FailureReason,
		FailureCode:       a.FailureCode,
		TriggeredBy:       a.TriggeredBy,
		AttemptNumber:     a.AttemptNumber,
		Amount:            a.Amount,
		ProcessingTimeMs:  a.ProcessingTimeMs,
		PaymentMethodID:   a.PaymentMethodId,
		AttemptedAt:       timeOrZero(a.AttemptedAt),
		CompletedAt:       timeOrZero(a.CompletedAt),
		CreatedAt:         timeOrZero(a.CreatedAt),
		Metadata:          a.Metadata,
	}
}

type DunningCommunicationResponse struct {
	ID                string    `json:"id"`
	DunningCampaignID string    `json:"dunning_campaign_id"`
	CustomerID        string    `json:"customer_id"`
	Channel           string    `json:"channel"`
	TemplateID        string    `json:"template_id"`
	Provider          string    `json:"provider"`
	Status            string    `json:"status"`
	FailureReason     string    `json:"failure_reason,omitempty"`
	AttemptNumber     int       `json:"attempt_number"`
	Subject           string    `json:"subject,omitempty"`
	ContentPreview    string    `json:"content_preview,omitempty"`
	SentAt            time.Time `json:"sent_at,omitzero"`
	DeliveredAt       time.Time `json:"delivered_at,omitzero"`
	OpenedAt          time.Time `json:"opened_at,omitzero"`
	ClickedAt         time.Time `json:"clicked_at,omitzero"`
	BouncedAt         time.Time `json:"bounced_at,omitzero"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewDunningCommunicationResponse(c domain.DunningCommunication) DunningCommunicationResponse {
	return DunningCommunicationResponse{
		ID:                c.Id,
		DunningCampaignID: c.DunningCampaignId,
		CustomerID:        c.CustomerId,
		Channel:           string(c.Channel),
		TemplateID:        c.TemplateId,
		Provider:          c.Provider,
		Status:            string(c.Status),
		FailureReason:     c.FailureReason,
		AttemptNumber:     c.AttemptNumber,
		Subject:           c.Subject,
		ContentPreview:    c.ContentPreview,
		SentAt:            timeOrZero(c.SentAt),
		DeliveredAt:       timeOrZero(c.DeliveredAt),
		OpenedAt:          timeOrZero(c.OpenedAt),
		ClickedAt:         timeOrZero(c.ClickedAt),
		BouncedAt:         timeOrZero(c.BouncedAt),
		CreatedAt:         timeOrZero(c.CreatedAt),
		UpdatedAt:         timeOrZero(c.UpdatedAt),
	}
}

type PaymentUpdateTokenResponse struct {
	TokenID           string          `json:"token_id"`
	SubscriptionID    string          `json:"subscription_id"`
	CustomerID        string          `json:"customer_id"`
	DunningCampaignID string          `json:"dunning_campaign_id,omitempty"`
	Status            string          `json:"status"`
	ExpiresAt         time.Time       `json:"expires_at"`
	CreatedAt         time.Time       `json:"created_at"`
	LastUsedAt        time.Time       `json:"last_used_at,omitzero"`
	MaxUses           int             `json:"max_uses"`
	UsedCount         int             `json:"used_count"`
	AllowedActions    map[string]bool `json:"allowed_actions,omitempty"`
	AdminGenerated    bool            `json:"admin_generated"`
	CreatedBy         string          `json:"created_by,omitempty"`
}

func NewPaymentUpdateTokenResponse(t domain.PaymentUpdateToken) PaymentUpdateTokenResponse {
	return PaymentUpdateTokenResponse{
		TokenID:           t.TokenId,
		SubscriptionID:    t.SubscriptionId,
		CustomerID:        t.CustomerId,
		DunningCampaignID: t.DunningCampaignId,
		Status:            string(t.Status),
		ExpiresAt:         timeOrZero(t.ExpiresAt),
		CreatedAt:         timeOrZero(t.CreatedAt),
		LastUsedAt:        timeOrZero(t.LastUsedAt),
		MaxUses:           t.MaxUses,
		UsedCount:         t.UsedCount,
		AllowedActions:    t.AllowedActions,
		AdminGenerated:    t.AdminGenerated,
		CreatedBy:         t.CreatedBy,
	}
}

type DunningConfigurationResponse struct {
	ID               string                    `json:"id"`
	Name             string                    `json:"name"`
	Description      string                    `json:"description,omitempty"`
	Status           string                    `json:"status"`
	CreatedBy        string                    `json:"created_by,omitempty"`
	Priority         int                       `json:"priority"`
	AppliesTo        domain.DunningConfigScope `json:"applies_to"`
	TargetRules      map[string]any            `json:"target_rules,omitempty"`
	Config           map[string]any            `json:"config"`
	IsAbTest         bool                      `json:"is_ab_test"`
	AbTestPercentage float64                   `json:"ab_test_percentage,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
}

func NewDunningConfigurationResponse(c domain.DunningConfiguration) DunningConfigurationResponse {
	return DunningConfigurationResponse{
		ID:               c.Id,
		Name:             c.Name,
		Description:      c.Description,
		Status:           string(c.Status),
		CreatedBy:        c.CreatedBy,
		Priority:         c.Priority,
		AppliesTo:        c.AppliesTo,
		TargetRules:      c.TargetRules,
		Config:           c.Config,
		IsAbTest:         c.IsAbTest,
		AbTestPercentage: c.AbTestPercentage,
		CreatedAt:        timeOrZero(c.CreatedAt),
		UpdatedAt:        timeOrZero(c.UpdatedAt),
	}
}

type CustomerDunningHistoryResponse struct {
	CustomerID              string    `json:"customer_id"`
	PreferredRecoveryMethod string    `json:"preferred_recovery_method,omitempty"`
	DunningRiskTier         string    `json:"dunning_risk_tier,omitempty"`
	TotalDunningCampaigns   int       `json:"total_dunning_campaigns"`
	SuccessfulRecoveries    int       `json:"successful_recoveries"`
	FailedCampaigns         int       `json:"failed_campaigns"`
	TotalAmountAtRisk       int64     `json:"total_amount_at_risk"`
	TotalAmountRecovered    int64     `json:"total_amount_recovered"`
	TotalAmountLost         int64     `json:"total_amount_lost"`
	AvgRecoveryTimeHours    float64   `json:"avg_recovery_time_hours,omitempty"`
	PaymentReliabilityScore float64   `json:"payment_reliability_score,omitempty"`
	MostResponsiveChannel   string    `json:"most_responsive_channel,omitempty"`
	FirstDunningAt          time.Time `json:"first_dunning_at,omitzero"`
	LastDunningAt           time.Time `json:"last_dunning_at,omitzero"`
	LastRecoveryAt          time.Time `json:"last_recovery_at,omitzero"`
}

func NewCustomerDunningHistoryResponse(h domain.CustomerDunningHistory) CustomerDunningHistoryResponse {
	return CustomerDunningHistoryResponse{
		CustomerID:              h.CustomerId,
		PreferredRecoveryMethod: h.PreferredRecoveryMethod,
		DunningRiskTier:         h.DunningRiskTier,
		TotalDunningCampaigns:   h.TotalDunningCampaigns,
		SuccessfulRecoveries:    h.SuccessfulRecoveries,
		FailedCampaigns:         h.FailedCampaigns,
		TotalAmountAtRisk:       h.TotalAmountAtRisk,
		TotalAmountRecovered:    h.TotalAmountRecovered,
		TotalAmountLost:         h.TotalAmountLost,
		AvgRecoveryTimeHours:    h.AvgRecoveryTimeHours,
		PaymentReliabilityScore: h.PaymentReliabilityScore,
		MostResponsiveChannel:   string(h.MostResponsiveChannel),
		FirstDunningAt:          timeOrZero(h.FirstDunningAt),
		LastDunningAt:           timeOrZero(h.LastDunningAt),
		LastRecoveryAt:          timeOrZero(h.LastRecoveryAt),
	}
}
