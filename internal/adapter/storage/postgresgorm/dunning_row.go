package postgresgorm

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// dunningCampaignRow is the postgres on-the-wire shape of a DunningCampaign.
type dunningCampaignRow struct {
	OrgId            string `gorm:"column:org_id;primaryKey"`
	Id               string `gorm:"column:id;primaryKey"`
	SubscriptionId   string `gorm:"column:subscription_id"`
	CustomerId       string `gorm:"column:customer_id"`
	WorkflowId       string `gorm:"column:workflow_id"`
	WorkflowRunId    string `gorm:"column:workflow_run_id"`
	ParentWorkflowId string `gorm:"column:parent_workflow_id"`

	Status               domain.DunningStatus `gorm:"column:status"`
	FailedAmount         int64                `gorm:"column:failed_amount"`
	Currency             string               `gorm:"column:currency"`
	InitialFailureReason string               `gorm:"column:initial_failure_reason"`

	TotalAttempts       int `gorm:"column:total_attempts"`
	ImmediateAttempts   int `gorm:"column:immediate_attempts"`
	ProgressiveAttempts int `gorm:"column:progressive_attempts"`

	StartedAt     time.Time `gorm:"column:started_at"`
	LastAttemptAt time.Time `gorm:"column:last_attempt_at"`
	NextAttemptAt time.Time `gorm:"column:next_attempt_at"`
	CompletedAt   time.Time `gorm:"column:completed_at"`

	RecoveryMethod     string    `gorm:"column:recovery_method"`
	RecoveredAmount    int64     `gorm:"column:recovered_amount"`
	RecoveredAt        time.Time `gorm:"column:recovered_at"`
	FinalFailureReason string    `gorm:"column:final_failure_reason"`

	ConfigSnapshot map[string]any    `gorm:"column:config_snapshot;serializer:json"`
	Metadata       map[string]string `gorm:"column:metadata;serializer:json"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (dunningCampaignRow) TableName() string { return "dunning_campaigns" }

func (r dunningCampaignRow) toDomain() domain.DunningCampaign {
	return domain.DunningCampaign{
		OrgId:                r.OrgId,
		Id:                   r.Id,
		SubscriptionId:       r.SubscriptionId,
		CustomerId:           r.CustomerId,
		WorkflowId:           r.WorkflowId,
		WorkflowRunId:        r.WorkflowRunId,
		ParentWorkflowId:     r.ParentWorkflowId,
		Status:               r.Status,
		FailedAmount:         r.FailedAmount,
		Currency:             r.Currency,
		InitialFailureReason: r.InitialFailureReason,
		TotalAttempts:        r.TotalAttempts,
		ImmediateAttempts:    r.ImmediateAttempts,
		ProgressiveAttempts:  r.ProgressiveAttempts,
		StartedAt:            r.StartedAt,
		LastAttemptAt:        r.LastAttemptAt,
		NextAttemptAt:        r.NextAttemptAt,
		CompletedAt:          r.CompletedAt,
		RecoveryMethod:       r.RecoveryMethod,
		RecoveredAmount:      r.RecoveredAmount,
		RecoveredAt:          r.RecoveredAt,
		FinalFailureReason:   r.FinalFailureReason,
		ConfigSnapshot:       r.ConfigSnapshot,
		Metadata:             r.Metadata,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func dunningCampaignRowFromDomain(c domain.DunningCampaign) dunningCampaignRow {
	return dunningCampaignRow{
		OrgId:                c.OrgId,
		Id:                   c.Id,
		SubscriptionId:       c.SubscriptionId,
		CustomerId:           c.CustomerId,
		WorkflowId:           c.WorkflowId,
		WorkflowRunId:        c.WorkflowRunId,
		ParentWorkflowId:     c.ParentWorkflowId,
		Status:               c.Status,
		FailedAmount:         c.FailedAmount,
		Currency:             c.Currency,
		InitialFailureReason: c.InitialFailureReason,
		TotalAttempts:        c.TotalAttempts,
		ImmediateAttempts:    c.ImmediateAttempts,
		ProgressiveAttempts:  c.ProgressiveAttempts,
		StartedAt:            c.StartedAt,
		LastAttemptAt:        c.LastAttemptAt,
		NextAttemptAt:        c.NextAttemptAt,
		CompletedAt:          c.CompletedAt,
		RecoveryMethod:       c.RecoveryMethod,
		RecoveredAmount:      c.RecoveredAmount,
		RecoveredAt:          c.RecoveredAt,
		FinalFailureReason:   c.FinalFailureReason,
		ConfigSnapshot:       c.ConfigSnapshot,
		Metadata:             c.Metadata,
		CreatedAt:            c.CreatedAt,
		UpdatedAt:            c.UpdatedAt,
	}
}

func dunningCampaignRowsToDomain(rows []dunningCampaignRow) []domain.DunningCampaign {
	out := make([]domain.DunningCampaign, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}

// dunningAttemptRow is the postgres on-the-wire shape of a DunningAttempt.
type dunningAttemptRow struct {
	OrgId             string                    `gorm:"column:org_id;primaryKey"`
	Id                string                    `gorm:"column:id;primaryKey"`
	DunningCampaignId string                    `gorm:"column:dunning_campaign_id"`
	SubscriptionId    string                    `gorm:"column:subscription_id"`
	AttemptNumber     int                       `gorm:"column:attempt_number"`
	AttemptType       domain.DunningAttemptType `gorm:"column:attempt_type"`
	Amount            int64                     `gorm:"column:amount"`
	Currency          string                    `gorm:"column:currency"`
	PaymentMethodId   string                    `gorm:"column:payment_method_id"`
	Status            domain.PaymentStatus      `gorm:"column:status"`
	FailureReason     string                    `gorm:"column:failure_reason"`
	FailureCode       string                    `gorm:"column:failure_code"`
	ProcessorResponse map[string]any            `gorm:"column:processor_response;serializer:json"`
	ProcessingTimeMs  int                       `gorm:"column:processing_time_ms"`
	AttemptedAt       time.Time                 `gorm:"column:attempted_at"`
	CompletedAt       time.Time                 `gorm:"column:completed_at"`
	TriggeredBy       string                    `gorm:"column:triggered_by"`
	Metadata          map[string]string         `gorm:"column:metadata;serializer:json"`
	CreatedAt         time.Time                 `gorm:"column:created_at"`
}

func (dunningAttemptRow) TableName() string { return "dunning_attempts" }

func (r dunningAttemptRow) toDomain() domain.DunningAttempt {
	return domain.DunningAttempt{
		OrgId:             r.OrgId,
		Id:                r.Id,
		DunningCampaignId: r.DunningCampaignId,
		SubscriptionId:    r.SubscriptionId,
		AttemptNumber:     r.AttemptNumber,
		AttemptType:       r.AttemptType,
		Amount:            r.Amount,
		Currency:          r.Currency,
		PaymentMethodId:   r.PaymentMethodId,
		Status:            r.Status,
		FailureReason:     r.FailureReason,
		FailureCode:       r.FailureCode,
		ProcessorResponse: r.ProcessorResponse,
		ProcessingTimeMs:  r.ProcessingTimeMs,
		AttemptedAt:       r.AttemptedAt,
		CompletedAt:       r.CompletedAt,
		TriggeredBy:       r.TriggeredBy,
		Metadata:          r.Metadata,
		CreatedAt:         r.CreatedAt,
	}
}

func dunningAttemptRowFromDomain(a domain.DunningAttempt) dunningAttemptRow {
	return dunningAttemptRow{
		OrgId:             a.OrgId,
		Id:                a.Id,
		DunningCampaignId: a.DunningCampaignId,
		SubscriptionId:    a.SubscriptionId,
		AttemptNumber:     a.AttemptNumber,
		AttemptType:       a.AttemptType,
		Amount:            a.Amount,
		Currency:          a.Currency,
		PaymentMethodId:   a.PaymentMethodId,
		Status:            a.Status,
		FailureReason:     a.FailureReason,
		FailureCode:       a.FailureCode,
		ProcessorResponse: a.ProcessorResponse,
		ProcessingTimeMs:  a.ProcessingTimeMs,
		AttemptedAt:       a.AttemptedAt,
		CompletedAt:       a.CompletedAt,
		TriggeredBy:       a.TriggeredBy,
		Metadata:          a.Metadata,
		CreatedAt:         a.CreatedAt,
	}
}

func dunningAttemptRowsToDomain(rows []dunningAttemptRow) []domain.DunningAttempt {
	out := make([]domain.DunningAttempt, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}

// dunningCommunicationRow is the postgres on-the-wire shape of a DunningCommunication.
type dunningCommunicationRow struct {
	OrgId               string                      `gorm:"column:org_id;primaryKey"`
	Id                  string                      `gorm:"column:id;primaryKey"`
	DunningCampaignId   string                      `gorm:"column:dunning_campaign_id"`
	CustomerId          string                      `gorm:"column:customer_id"`
	Channel             domain.CommunicationChannel `gorm:"column:channel"`
	TemplateId          string                      `gorm:"column:template_id"`
	AttemptNumber       int                         `gorm:"column:attempt_number"`
	Subject             string                      `gorm:"column:subject"`
	ContentPreview      string                      `gorm:"column:content_preview"`
	PersonalizationData map[string]any              `gorm:"column:personalization_data;serializer:json"`
	SentAt              time.Time                   `gorm:"column:sent_at"`
	DeliveredAt         time.Time                   `gorm:"column:delivered_at"`
	OpenedAt            time.Time                   `gorm:"column:opened_at"`
	ClickedAt           time.Time                   `gorm:"column:clicked_at"`
	BouncedAt           time.Time                   `gorm:"column:bounced_at"`
	Provider            string                      `gorm:"column:provider"`
	ProviderMessageId   string                      `gorm:"column:provider_message_id"`
	ProviderResponse    map[string]any              `gorm:"column:provider_response;serializer:json"`
	Status              domain.CommunicationStatus  `gorm:"column:status"`
	FailureReason       string                      `gorm:"column:failure_reason"`
	CreatedAt           time.Time                   `gorm:"column:created_at"`
	UpdatedAt           time.Time                   `gorm:"column:updated_at"`
}

func (dunningCommunicationRow) TableName() string { return "dunning_communications" }

func (r dunningCommunicationRow) toDomain() domain.DunningCommunication {
	return domain.DunningCommunication{
		OrgId:               r.OrgId,
		Id:                  r.Id,
		DunningCampaignId:   r.DunningCampaignId,
		CustomerId:          r.CustomerId,
		Channel:             r.Channel,
		TemplateId:          r.TemplateId,
		AttemptNumber:       r.AttemptNumber,
		Subject:             r.Subject,
		ContentPreview:      r.ContentPreview,
		PersonalizationData: r.PersonalizationData,
		SentAt:              r.SentAt,
		DeliveredAt:         r.DeliveredAt,
		OpenedAt:            r.OpenedAt,
		ClickedAt:           r.ClickedAt,
		BouncedAt:           r.BouncedAt,
		Provider:            r.Provider,
		ProviderMessageId:   r.ProviderMessageId,
		ProviderResponse:    r.ProviderResponse,
		Status:              r.Status,
		FailureReason:       r.FailureReason,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}

func dunningCommunicationRowFromDomain(c domain.DunningCommunication) dunningCommunicationRow {
	return dunningCommunicationRow{
		OrgId:               c.OrgId,
		Id:                  c.Id,
		DunningCampaignId:   c.DunningCampaignId,
		CustomerId:          c.CustomerId,
		Channel:             c.Channel,
		TemplateId:          c.TemplateId,
		AttemptNumber:       c.AttemptNumber,
		Subject:             c.Subject,
		ContentPreview:      c.ContentPreview,
		PersonalizationData: c.PersonalizationData,
		SentAt:              c.SentAt,
		DeliveredAt:         c.DeliveredAt,
		OpenedAt:            c.OpenedAt,
		ClickedAt:           c.ClickedAt,
		BouncedAt:           c.BouncedAt,
		Provider:            c.Provider,
		ProviderMessageId:   c.ProviderMessageId,
		ProviderResponse:    c.ProviderResponse,
		Status:              c.Status,
		FailureReason:       c.FailureReason,
		CreatedAt:           c.CreatedAt,
		UpdatedAt:           c.UpdatedAt,
	}
}

func dunningCommunicationRowsToDomain(rows []dunningCommunicationRow) []domain.DunningCommunication {
	out := make([]domain.DunningCommunication, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}

// paymentUpdateTokenRow is the postgres on-the-wire shape of a PaymentUpdateToken.
type paymentUpdateTokenRow struct {
	OrgId             string             `gorm:"column:org_id;primaryKey"`
	TokenId           string             `gorm:"column:token_id;primaryKey"`
	SubscriptionId    string             `gorm:"column:subscription_id"`
	CustomerId        string             `gorm:"column:customer_id"`
	DunningCampaignId string             `gorm:"column:dunning_campaign_id"`
	TokenData         map[string]any     `gorm:"column:token_data;serializer:json"`
	Signature         string             `gorm:"column:signature"`
	ExpiresAt         time.Time          `gorm:"column:expires_at"`
	MaxUses           int                `gorm:"column:max_uses"`
	UsedCount         int                `gorm:"column:used_count"`
	Status            domain.TokenStatus `gorm:"column:status"`
	AllowedActions    map[string]bool    `gorm:"column:allowed_actions;serializer:json"`
	AdminGenerated    bool               `gorm:"column:admin_generated"`
	AdminUserId       string             `gorm:"column:admin_user_id"`
	AdminReason       string             `gorm:"column:admin_reason"`
	AdminNotes        string             `gorm:"column:admin_notes"`
	CreatedBy         string             `gorm:"column:created_by"`
	CreatedAt         time.Time          `gorm:"column:created_at"`
	LastUsedAt        time.Time          `gorm:"column:last_used_at"`
	LastUsedIp        string             `gorm:"column:last_used_ip"`
}

func (paymentUpdateTokenRow) TableName() string { return "payment_update_tokens" }

func (r paymentUpdateTokenRow) toDomain() domain.PaymentUpdateToken {
	return domain.PaymentUpdateToken{
		OrgId:             r.OrgId,
		TokenId:           r.TokenId,
		SubscriptionId:    r.SubscriptionId,
		CustomerId:        r.CustomerId,
		DunningCampaignId: r.DunningCampaignId,
		TokenData:         r.TokenData,
		Signature:         r.Signature,
		ExpiresAt:         r.ExpiresAt,
		MaxUses:           r.MaxUses,
		UsedCount:         r.UsedCount,
		Status:            r.Status,
		AllowedActions:    r.AllowedActions,
		AdminGenerated:    r.AdminGenerated,
		AdminUserId:       r.AdminUserId,
		AdminReason:       r.AdminReason,
		AdminNotes:        r.AdminNotes,
		CreatedBy:         r.CreatedBy,
		CreatedAt:         r.CreatedAt,
		LastUsedAt:        r.LastUsedAt,
		LastUsedIp:        r.LastUsedIp,
	}
}

func paymentUpdateTokenRowFromDomain(t domain.PaymentUpdateToken) paymentUpdateTokenRow {
	return paymentUpdateTokenRow{
		OrgId:             t.OrgId,
		TokenId:           t.TokenId,
		SubscriptionId:    t.SubscriptionId,
		CustomerId:        t.CustomerId,
		DunningCampaignId: t.DunningCampaignId,
		TokenData:         t.TokenData,
		Signature:         t.Signature,
		ExpiresAt:         t.ExpiresAt,
		MaxUses:           t.MaxUses,
		UsedCount:         t.UsedCount,
		Status:            t.Status,
		AllowedActions:    t.AllowedActions,
		AdminGenerated:    t.AdminGenerated,
		AdminUserId:       t.AdminUserId,
		AdminReason:       t.AdminReason,
		AdminNotes:        t.AdminNotes,
		CreatedBy:         t.CreatedBy,
		CreatedAt:         t.CreatedAt,
		LastUsedAt:        t.LastUsedAt,
		LastUsedIp:        t.LastUsedIp,
	}
}

func paymentUpdateTokenRowsToDomain(rows []paymentUpdateTokenRow) []domain.PaymentUpdateToken {
	out := make([]domain.PaymentUpdateToken, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}

// dunningConfigurationRow is the postgres on-the-wire shape of a DunningConfiguration.
type dunningConfigurationRow struct {
	OrgId            string                    `gorm:"column:org_id;primaryKey"`
	Id               string                    `gorm:"column:id;primaryKey"`
	Name             string                    `gorm:"column:name"`
	Description      string                    `gorm:"column:description"`
	Priority         int                       `gorm:"column:priority"`
	AppliesTo        domain.DunningConfigScope `gorm:"column:applies_to"`
	TargetRules      map[string]any            `gorm:"column:target_rules;serializer:json"`
	Config           map[string]any            `gorm:"column:config;serializer:json"`
	Status           domain.ConfigStatus       `gorm:"column:status"`
	IsAbTest         bool                      `gorm:"column:is_ab_test"`
	AbTestPercentage float64                   `gorm:"column:ab_test_percentage"`
	CreatedBy        string                    `gorm:"column:created_by"`
	CreatedAt        time.Time                 `gorm:"column:created_at"`
	UpdatedAt        time.Time                 `gorm:"column:updated_at"`
}

func (dunningConfigurationRow) TableName() string { return "dunning_configurations" }

func (r dunningConfigurationRow) toDomain() domain.DunningConfiguration {
	return domain.DunningConfiguration{
		OrgId:            r.OrgId,
		Id:               r.Id,
		Name:             r.Name,
		Description:      r.Description,
		Priority:         r.Priority,
		AppliesTo:        r.AppliesTo,
		TargetRules:      r.TargetRules,
		Config:           r.Config,
		Status:           r.Status,
		IsAbTest:         r.IsAbTest,
		AbTestPercentage: r.AbTestPercentage,
		CreatedBy:        r.CreatedBy,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func dunningConfigurationRowFromDomain(c domain.DunningConfiguration) dunningConfigurationRow {
	return dunningConfigurationRow{
		OrgId:            c.OrgId,
		Id:               c.Id,
		Name:             c.Name,
		Description:      c.Description,
		Priority:         c.Priority,
		AppliesTo:        c.AppliesTo,
		TargetRules:      c.TargetRules,
		Config:           c.Config,
		Status:           c.Status,
		IsAbTest:         c.IsAbTest,
		AbTestPercentage: c.AbTestPercentage,
		CreatedBy:        c.CreatedBy,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func dunningConfigurationRowsToDomain(rows []dunningConfigurationRow) []domain.DunningConfiguration {
	out := make([]domain.DunningConfiguration, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}

// customerDunningHistoryRow is the postgres on-the-wire shape of a CustomerDunningHistory.
type customerDunningHistoryRow struct {
	OrgId      string `gorm:"column:org_id;primaryKey"`
	CustomerId string `gorm:"column:customer_id;primaryKey"`

	TotalDunningCampaigns int `gorm:"column:total_dunning_campaigns"`
	SuccessfulRecoveries  int `gorm:"column:successful_recoveries"`
	FailedCampaigns       int `gorm:"column:failed_campaigns"`

	TotalAmountAtRisk    int64 `gorm:"column:total_amount_at_risk"`
	TotalAmountRecovered int64 `gorm:"column:total_amount_recovered"`
	TotalAmountLost      int64 `gorm:"column:total_amount_lost"`

	AvgRecoveryTimeHours    float64                     `gorm:"column:avg_recovery_time_hours"`
	PreferredRecoveryMethod string                      `gorm:"column:preferred_recovery_method"`
	MostResponsiveChannel   domain.CommunicationChannel `gorm:"column:most_responsive_channel"`
	PaymentReliabilityScore float64                     `gorm:"column:payment_reliability_score"`
	DunningRiskTier         string                      `gorm:"column:dunning_risk_tier"`

	FirstDunningAt time.Time `gorm:"column:first_dunning_at"`
	LastDunningAt  time.Time `gorm:"column:last_dunning_at"`
	LastRecoveryAt time.Time `gorm:"column:last_recovery_at"`

	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (customerDunningHistoryRow) TableName() string { return "customer_dunning_history" }

func (r customerDunningHistoryRow) toDomain() domain.CustomerDunningHistory {
	return domain.CustomerDunningHistory{
		OrgId:                   r.OrgId,
		CustomerId:              r.CustomerId,
		TotalDunningCampaigns:   r.TotalDunningCampaigns,
		SuccessfulRecoveries:    r.SuccessfulRecoveries,
		FailedCampaigns:         r.FailedCampaigns,
		TotalAmountAtRisk:       r.TotalAmountAtRisk,
		TotalAmountRecovered:    r.TotalAmountRecovered,
		TotalAmountLost:         r.TotalAmountLost,
		AvgRecoveryTimeHours:    r.AvgRecoveryTimeHours,
		PreferredRecoveryMethod: r.PreferredRecoveryMethod,
		MostResponsiveChannel:   r.MostResponsiveChannel,
		PaymentReliabilityScore: r.PaymentReliabilityScore,
		DunningRiskTier:         r.DunningRiskTier,
		FirstDunningAt:          r.FirstDunningAt,
		LastDunningAt:           r.LastDunningAt,
		LastRecoveryAt:          r.LastRecoveryAt,
		UpdatedAt:               r.UpdatedAt,
	}
}

func customerDunningHistoryRowFromDomain(h domain.CustomerDunningHistory) customerDunningHistoryRow {
	return customerDunningHistoryRow{
		OrgId:                   h.OrgId,
		CustomerId:              h.CustomerId,
		TotalDunningCampaigns:   h.TotalDunningCampaigns,
		SuccessfulRecoveries:    h.SuccessfulRecoveries,
		FailedCampaigns:         h.FailedCampaigns,
		TotalAmountAtRisk:       h.TotalAmountAtRisk,
		TotalAmountRecovered:    h.TotalAmountRecovered,
		TotalAmountLost:         h.TotalAmountLost,
		AvgRecoveryTimeHours:    h.AvgRecoveryTimeHours,
		PreferredRecoveryMethod: h.PreferredRecoveryMethod,
		MostResponsiveChannel:   h.MostResponsiveChannel,
		PaymentReliabilityScore: h.PaymentReliabilityScore,
		DunningRiskTier:         h.DunningRiskTier,
		FirstDunningAt:          h.FirstDunningAt,
		LastDunningAt:           h.LastDunningAt,
		LastRecoveryAt:          h.LastRecoveryAt,
		UpdatedAt:               h.UpdatedAt,
	}
}
