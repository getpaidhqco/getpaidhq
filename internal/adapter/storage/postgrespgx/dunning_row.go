package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// ---- Campaigns ----

// dunningCampaignRow is the postgres on-the-wire shape of a DunningCampaign.
// Nullable text columns (parent_workflow_id, initial_failure_reason,
// recovery_method, final_failure_reason) and the nullable recovered_amount are
// held as pointers; the nullable timestamps (last_attempt_at, next_attempt_at,
// completed_at, recovered_at) follow the nulltime convention via *time.Time.
type dunningCampaignRow struct {
	OrgId            string
	Id               string
	SubscriptionId   string
	CustomerId       string
	WorkflowId       string
	WorkflowRunId    string
	ParentWorkflowId *string

	Status               string
	FailedAmount         int64
	Currency             string
	InitialFailureReason *string

	TotalAttempts       int
	ImmediateAttempts   int
	ProgressiveAttempts int

	StartedAt     time.Time
	LastAttemptAt *time.Time
	NextAttemptAt *time.Time
	CompletedAt   *time.Time

	RecoveryMethod     *string
	RecoveredAmount    *int64
	RecoveredAt        *time.Time
	FinalFailureReason *string

	ConfigSnapshot jsonCol[map[string]any]
	Metadata       jsonCol[map[string]string]

	CreatedAt time.Time
	UpdatedAt time.Time
}

const dunningCampaignColumns = `org_id, id, subscription_id, customer_id, workflow_id, workflow_run_id, parent_workflow_id, ` +
	`status, failed_amount, currency, initial_failure_reason, ` +
	`total_attempts, immediate_attempts, progressive_attempts, ` +
	`started_at, last_attempt_at, next_attempt_at, completed_at, ` +
	`recovery_method, recovered_amount, recovered_at, final_failure_reason, ` +
	`config_snapshot, metadata, created_at, updated_at`

func (r *dunningCampaignRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.SubscriptionId, &r.CustomerId, &r.WorkflowId, &r.WorkflowRunId, &r.ParentWorkflowId,
		&r.Status, &r.FailedAmount, &r.Currency, &r.InitialFailureReason,
		&r.TotalAttempts, &r.ImmediateAttempts, &r.ProgressiveAttempts,
		&r.StartedAt, &r.LastAttemptAt, &r.NextAttemptAt, &r.CompletedAt,
		&r.RecoveryMethod, &r.RecoveredAmount, &r.RecoveredAt, &r.FinalFailureReason,
		&r.ConfigSnapshot, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r dunningCampaignRow) toDomain() domain.DunningCampaign {
	var recoveredAmount int64
	if r.RecoveredAmount != nil {
		recoveredAmount = *r.RecoveredAmount
	}
	return domain.DunningCampaign{
		OrgId:                r.OrgId,
		Id:                   r.Id,
		SubscriptionId:       r.SubscriptionId,
		CustomerId:           r.CustomerId,
		WorkflowId:           r.WorkflowId,
		WorkflowRunId:        r.WorkflowRunId,
		ParentWorkflowId:     strOrEmpty(r.ParentWorkflowId),
		Status:               domain.DunningStatus(r.Status),
		FailedAmount:         r.FailedAmount,
		Currency:             r.Currency,
		InitialFailureReason: strOrEmpty(r.InitialFailureReason),
		TotalAttempts:        r.TotalAttempts,
		ImmediateAttempts:    r.ImmediateAttempts,
		ProgressiveAttempts:  r.ProgressiveAttempts,
		StartedAt:            r.StartedAt,
		LastAttemptAt:        timeOrZero(r.LastAttemptAt),
		NextAttemptAt:        timeOrZero(r.NextAttemptAt),
		CompletedAt:          timeOrZero(r.CompletedAt),
		RecoveryMethod:       strOrEmpty(r.RecoveryMethod),
		RecoveredAmount:      recoveredAmount,
		RecoveredAt:          timeOrZero(r.RecoveredAt),
		FinalFailureReason:   strOrEmpty(r.FinalFailureReason),
		ConfigSnapshot:       r.ConfigSnapshot.V,
		Metadata:             r.Metadata.V,
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
		ParentWorkflowId:     nilIfEmpty(c.ParentWorkflowId),
		Status:               string(c.Status),
		FailedAmount:         c.FailedAmount,
		Currency:             c.Currency,
		InitialFailureReason: nilIfEmpty(c.InitialFailureReason),
		TotalAttempts:        c.TotalAttempts,
		ImmediateAttempts:    c.ImmediateAttempts,
		ProgressiveAttempts:  c.ProgressiveAttempts,
		StartedAt:            c.StartedAt,
		LastAttemptAt:        nullTime(c.LastAttemptAt),
		NextAttemptAt:        nullTime(c.NextAttemptAt),
		CompletedAt:          nullTime(c.CompletedAt),
		RecoveryMethod:       nilIfEmpty(c.RecoveryMethod),
		RecoveredAmount:      &c.RecoveredAmount,
		RecoveredAt:          nullTime(c.RecoveredAt),
		FinalFailureReason:   nilIfEmpty(c.FinalFailureReason),
		ConfigSnapshot:       newJSON(c.ConfigSnapshot),
		Metadata:             newJSON(c.Metadata),
		CreatedAt:            c.CreatedAt,
		UpdatedAt:            c.UpdatedAt,
	}
}

// ---- Attempts ----

// dunningAttemptRow is the postgres on-the-wire shape of a DunningAttempt.
type dunningAttemptRow struct {
	OrgId             string
	Id                string
	DunningCampaignId string
	SubscriptionId    string
	AttemptNumber     int
	AttemptType       string
	Amount            int64
	Currency          string
	PaymentMethodId   *string
	Status            string
	FailureReason     *string
	FailureCode       *string
	ProcessorResponse jsonCol[map[string]any]
	ProcessingTimeMs  *int
	AttemptedAt       time.Time
	CompletedAt       *time.Time
	TriggeredBy       *string
	Metadata          jsonCol[map[string]string]
	CreatedAt         time.Time
}

const dunningAttemptColumns = `org_id, id, dunning_campaign_id, subscription_id, attempt_number, attempt_type, ` +
	`amount, currency, payment_method_id, status, failure_reason, failure_code, ` +
	`processor_response, processing_time_ms, attempted_at, completed_at, triggered_by, metadata, created_at`

func (r *dunningAttemptRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.DunningCampaignId, &r.SubscriptionId, &r.AttemptNumber, &r.AttemptType,
		&r.Amount, &r.Currency, &r.PaymentMethodId, &r.Status, &r.FailureReason, &r.FailureCode,
		&r.ProcessorResponse, &r.ProcessingTimeMs, &r.AttemptedAt, &r.CompletedAt, &r.TriggeredBy, &r.Metadata, &r.CreatedAt)
}

func (r dunningAttemptRow) toDomain() domain.DunningAttempt {
	var processingTimeMs int
	if r.ProcessingTimeMs != nil {
		processingTimeMs = *r.ProcessingTimeMs
	}
	return domain.DunningAttempt{
		OrgId:             r.OrgId,
		Id:                r.Id,
		DunningCampaignId: r.DunningCampaignId,
		SubscriptionId:    r.SubscriptionId,
		AttemptNumber:     r.AttemptNumber,
		AttemptType:       domain.DunningAttemptType(r.AttemptType),
		Amount:            r.Amount,
		Currency:          r.Currency,
		PaymentMethodId:   strOrEmpty(r.PaymentMethodId),
		Status:            domain.PaymentStatus(r.Status),
		FailureReason:     strOrEmpty(r.FailureReason),
		FailureCode:       strOrEmpty(r.FailureCode),
		ProcessorResponse: r.ProcessorResponse.V,
		ProcessingTimeMs:  processingTimeMs,
		AttemptedAt:       r.AttemptedAt,
		CompletedAt:       timeOrZero(r.CompletedAt),
		TriggeredBy:       strOrEmpty(r.TriggeredBy),
		Metadata:          r.Metadata.V,
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
		AttemptType:       string(a.AttemptType),
		Amount:            a.Amount,
		Currency:          a.Currency,
		PaymentMethodId:   nilIfEmpty(a.PaymentMethodId),
		Status:            string(a.Status),
		FailureReason:     nilIfEmpty(a.FailureReason),
		FailureCode:       nilIfEmpty(a.FailureCode),
		ProcessorResponse: newJSON(a.ProcessorResponse),
		ProcessingTimeMs:  &a.ProcessingTimeMs,
		AttemptedAt:       a.AttemptedAt,
		CompletedAt:       nullTime(a.CompletedAt),
		TriggeredBy:       nilIfEmpty(a.TriggeredBy),
		Metadata:          newJSON(a.Metadata),
		CreatedAt:         a.CreatedAt,
	}
}

// ---- Communications ----

// dunningCommunicationRow is the postgres on-the-wire shape of a DunningCommunication.
type dunningCommunicationRow struct {
	OrgId               string
	Id                  string
	DunningCampaignId   string
	CustomerId          string
	Channel             string
	TemplateId          string
	AttemptNumber       int
	Subject             *string
	ContentPreview      *string
	PersonalizationData jsonCol[map[string]any]
	SentAt              *time.Time
	DeliveredAt         *time.Time
	OpenedAt            *time.Time
	ClickedAt           *time.Time
	BouncedAt           *time.Time
	Provider            string
	ProviderMessageId   *string
	ProviderResponse    jsonCol[map[string]any]
	Status              string
	FailureReason       *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

const dunningCommunicationColumns = `org_id, id, dunning_campaign_id, customer_id, channel, template_id, attempt_number, ` +
	`subject, content_preview, personalization_data, ` +
	`sent_at, delivered_at, opened_at, clicked_at, bounced_at, ` +
	`provider, provider_message_id, provider_response, status, failure_reason, created_at, updated_at`

func (r *dunningCommunicationRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.DunningCampaignId, &r.CustomerId, &r.Channel, &r.TemplateId, &r.AttemptNumber,
		&r.Subject, &r.ContentPreview, &r.PersonalizationData,
		&r.SentAt, &r.DeliveredAt, &r.OpenedAt, &r.ClickedAt, &r.BouncedAt,
		&r.Provider, &r.ProviderMessageId, &r.ProviderResponse, &r.Status, &r.FailureReason, &r.CreatedAt, &r.UpdatedAt)
}

func (r dunningCommunicationRow) toDomain() domain.DunningCommunication {
	return domain.DunningCommunication{
		OrgId:               r.OrgId,
		Id:                  r.Id,
		DunningCampaignId:   r.DunningCampaignId,
		CustomerId:          r.CustomerId,
		Channel:             domain.CommunicationChannel(r.Channel),
		TemplateId:          r.TemplateId,
		AttemptNumber:       r.AttemptNumber,
		Subject:             strOrEmpty(r.Subject),
		ContentPreview:      strOrEmpty(r.ContentPreview),
		PersonalizationData: r.PersonalizationData.V,
		SentAt:              timeOrZero(r.SentAt),
		DeliveredAt:         timeOrZero(r.DeliveredAt),
		OpenedAt:            timeOrZero(r.OpenedAt),
		ClickedAt:           timeOrZero(r.ClickedAt),
		BouncedAt:           timeOrZero(r.BouncedAt),
		Provider:            r.Provider,
		ProviderMessageId:   strOrEmpty(r.ProviderMessageId),
		ProviderResponse:    r.ProviderResponse.V,
		Status:              domain.CommunicationStatus(r.Status),
		FailureReason:       strOrEmpty(r.FailureReason),
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
		Channel:             string(c.Channel),
		TemplateId:          c.TemplateId,
		AttemptNumber:       c.AttemptNumber,
		Subject:             nilIfEmpty(c.Subject),
		ContentPreview:      nilIfEmpty(c.ContentPreview),
		PersonalizationData: newJSON(c.PersonalizationData),
		SentAt:              nullTime(c.SentAt),
		DeliveredAt:         nullTime(c.DeliveredAt),
		OpenedAt:            nullTime(c.OpenedAt),
		ClickedAt:           nullTime(c.ClickedAt),
		BouncedAt:           nullTime(c.BouncedAt),
		Provider:            c.Provider,
		ProviderMessageId:   nilIfEmpty(c.ProviderMessageId),
		ProviderResponse:    newJSON(c.ProviderResponse),
		Status:              string(c.Status),
		FailureReason:       nilIfEmpty(c.FailureReason),
		CreatedAt:           c.CreatedAt,
		UpdatedAt:           c.UpdatedAt,
	}
}

// ---- Tokens ----

// paymentUpdateTokenRow is the postgres on-the-wire shape of a PaymentUpdateToken.
type paymentUpdateTokenRow struct {
	OrgId             string
	TokenId           string
	SubscriptionId    string
	CustomerId        string
	DunningCampaignId *string
	TokenData         jsonCol[map[string]any]
	Signature         string
	ExpiresAt         time.Time
	MaxUses           int
	UsedCount         int
	Status            string
	AllowedActions    jsonCol[map[string]bool]
	AdminGenerated    bool
	AdminUserId       *string
	AdminReason       *string
	AdminNotes        *string
	CreatedBy         *string
	CreatedAt         time.Time
	LastUsedAt        *time.Time
	LastUsedIp        *string
}

const paymentUpdateTokenColumns = `org_id, token_id, subscription_id, customer_id, dunning_campaign_id, ` +
	`token_data, signature, expires_at, max_uses, used_count, status, allowed_actions, ` +
	`admin_generated, admin_user_id, admin_reason, admin_notes, created_by, created_at, last_used_at, last_used_ip`

func (r *paymentUpdateTokenRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.TokenId, &r.SubscriptionId, &r.CustomerId, &r.DunningCampaignId,
		&r.TokenData, &r.Signature, &r.ExpiresAt, &r.MaxUses, &r.UsedCount, &r.Status, &r.AllowedActions,
		&r.AdminGenerated, &r.AdminUserId, &r.AdminReason, &r.AdminNotes, &r.CreatedBy, &r.CreatedAt, &r.LastUsedAt, &r.LastUsedIp)
}

func (r paymentUpdateTokenRow) toDomain() domain.PaymentUpdateToken {
	return domain.PaymentUpdateToken{
		OrgId:             r.OrgId,
		TokenId:           r.TokenId,
		SubscriptionId:    r.SubscriptionId,
		CustomerId:        r.CustomerId,
		DunningCampaignId: strOrEmpty(r.DunningCampaignId),
		TokenData:         r.TokenData.V,
		Signature:         r.Signature,
		ExpiresAt:         r.ExpiresAt,
		MaxUses:           r.MaxUses,
		UsedCount:         r.UsedCount,
		Status:            domain.TokenStatus(r.Status),
		AllowedActions:    r.AllowedActions.V,
		AdminGenerated:    r.AdminGenerated,
		AdminUserId:       strOrEmpty(r.AdminUserId),
		AdminReason:       strOrEmpty(r.AdminReason),
		AdminNotes:        strOrEmpty(r.AdminNotes),
		CreatedBy:         strOrEmpty(r.CreatedBy),
		CreatedAt:         r.CreatedAt,
		LastUsedAt:        timeOrZero(r.LastUsedAt),
		LastUsedIp:        strOrEmpty(r.LastUsedIp),
	}
}

func paymentUpdateTokenRowFromDomain(t domain.PaymentUpdateToken) paymentUpdateTokenRow {
	return paymentUpdateTokenRow{
		OrgId:             t.OrgId,
		TokenId:           t.TokenId,
		SubscriptionId:    t.SubscriptionId,
		CustomerId:        t.CustomerId,
		DunningCampaignId: nilIfEmpty(t.DunningCampaignId),
		TokenData:         newJSON(t.TokenData),
		Signature:         t.Signature,
		ExpiresAt:         t.ExpiresAt,
		MaxUses:           t.MaxUses,
		UsedCount:         t.UsedCount,
		Status:            string(t.Status),
		AllowedActions:    newJSON(t.AllowedActions),
		AdminGenerated:    t.AdminGenerated,
		AdminUserId:       nilIfEmpty(t.AdminUserId),
		AdminReason:       nilIfEmpty(t.AdminReason),
		AdminNotes:        nilIfEmpty(t.AdminNotes),
		CreatedBy:         nilIfEmpty(t.CreatedBy),
		CreatedAt:         t.CreatedAt,
		LastUsedAt:        nullTime(t.LastUsedAt),
		LastUsedIp:        nilIfEmpty(t.LastUsedIp),
	}
}

// ---- Configurations ----

// dunningConfigurationRow is the postgres on-the-wire shape of a DunningConfiguration.
type dunningConfigurationRow struct {
	OrgId            string
	Id               string
	Name             string
	Description      *string
	Priority         int
	AppliesTo        string
	TargetRules      jsonCol[map[string]any]
	Config           jsonCol[map[string]any]
	Status           string
	IsAbTest         bool
	AbTestPercentage *float64
	CreatedBy        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

const dunningConfigurationColumns = `org_id, id, name, description, priority, applies_to, target_rules, config, ` +
	`status, is_ab_test, ab_test_percentage, created_by, created_at, updated_at`

func (r *dunningConfigurationRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.Name, &r.Description, &r.Priority, &r.AppliesTo, &r.TargetRules, &r.Config,
		&r.Status, &r.IsAbTest, &r.AbTestPercentage, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
}

func (r dunningConfigurationRow) toDomain() domain.DunningConfiguration {
	var abTestPercentage float64
	if r.AbTestPercentage != nil {
		abTestPercentage = *r.AbTestPercentage
	}
	return domain.DunningConfiguration{
		OrgId:            r.OrgId,
		Id:               r.Id,
		Name:             r.Name,
		Description:      strOrEmpty(r.Description),
		Priority:         r.Priority,
		AppliesTo:        domain.DunningConfigScope(r.AppliesTo),
		TargetRules:      r.TargetRules.V,
		Config:           r.Config.V,
		Status:           domain.ConfigStatus(r.Status),
		IsAbTest:         r.IsAbTest,
		AbTestPercentage: abTestPercentage,
		CreatedBy:        strOrEmpty(r.CreatedBy),
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func dunningConfigurationRowFromDomain(c domain.DunningConfiguration) dunningConfigurationRow {
	return dunningConfigurationRow{
		OrgId:            c.OrgId,
		Id:               c.Id,
		Name:             c.Name,
		Description:      nilIfEmpty(c.Description),
		Priority:         c.Priority,
		AppliesTo:        string(c.AppliesTo),
		TargetRules:      newJSON(c.TargetRules),
		Config:           newJSON(c.Config),
		Status:           string(c.Status),
		IsAbTest:         c.IsAbTest,
		AbTestPercentage: &c.AbTestPercentage,
		CreatedBy:        nilIfEmpty(c.CreatedBy),
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

// ---- Customer history ----

// customerDunningHistoryRow is the postgres on-the-wire shape of a CustomerDunningHistory.
type customerDunningHistoryRow struct {
	OrgId      string
	CustomerId string

	TotalDunningCampaigns int
	SuccessfulRecoveries  int
	FailedCampaigns       int

	TotalAmountAtRisk    int64
	TotalAmountRecovered int64
	TotalAmountLost      int64

	AvgRecoveryTimeHours    *float64
	PreferredRecoveryMethod *string
	MostResponsiveChannel   *string
	PaymentReliabilityScore *float64
	DunningRiskTier         *string

	FirstDunningAt *time.Time
	LastDunningAt  *time.Time
	LastRecoveryAt *time.Time

	UpdatedAt time.Time
}

const customerDunningHistoryColumns = `org_id, customer_id, total_dunning_campaigns, successful_recoveries, failed_campaigns, ` +
	`total_amount_at_risk, total_amount_recovered, total_amount_lost, ` +
	`avg_recovery_time_hours, preferred_recovery_method, most_responsive_channel, payment_reliability_score, dunning_risk_tier, ` +
	`first_dunning_at, last_dunning_at, last_recovery_at, updated_at`

func (r *customerDunningHistoryRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.CustomerId, &r.TotalDunningCampaigns, &r.SuccessfulRecoveries, &r.FailedCampaigns,
		&r.TotalAmountAtRisk, &r.TotalAmountRecovered, &r.TotalAmountLost,
		&r.AvgRecoveryTimeHours, &r.PreferredRecoveryMethod, &r.MostResponsiveChannel, &r.PaymentReliabilityScore, &r.DunningRiskTier,
		&r.FirstDunningAt, &r.LastDunningAt, &r.LastRecoveryAt, &r.UpdatedAt)
}

func (r customerDunningHistoryRow) toDomain() domain.CustomerDunningHistory {
	var avgRecoveryTimeHours, paymentReliabilityScore float64
	if r.AvgRecoveryTimeHours != nil {
		avgRecoveryTimeHours = *r.AvgRecoveryTimeHours
	}
	if r.PaymentReliabilityScore != nil {
		paymentReliabilityScore = *r.PaymentReliabilityScore
	}
	return domain.CustomerDunningHistory{
		OrgId:                   r.OrgId,
		CustomerId:              r.CustomerId,
		TotalDunningCampaigns:   r.TotalDunningCampaigns,
		SuccessfulRecoveries:    r.SuccessfulRecoveries,
		FailedCampaigns:         r.FailedCampaigns,
		TotalAmountAtRisk:       r.TotalAmountAtRisk,
		TotalAmountRecovered:    r.TotalAmountRecovered,
		TotalAmountLost:         r.TotalAmountLost,
		AvgRecoveryTimeHours:    avgRecoveryTimeHours,
		PreferredRecoveryMethod: strOrEmpty(r.PreferredRecoveryMethod),
		MostResponsiveChannel:   domain.CommunicationChannel(strOrEmpty(r.MostResponsiveChannel)),
		PaymentReliabilityScore: paymentReliabilityScore,
		DunningRiskTier:         strOrEmpty(r.DunningRiskTier),
		FirstDunningAt:          timeOrZero(r.FirstDunningAt),
		LastDunningAt:           timeOrZero(r.LastDunningAt),
		LastRecoveryAt:          timeOrZero(r.LastRecoveryAt),
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
		AvgRecoveryTimeHours:    &h.AvgRecoveryTimeHours,
		PreferredRecoveryMethod: nilIfEmpty(h.PreferredRecoveryMethod),
		MostResponsiveChannel:   nilIfEmpty(string(h.MostResponsiveChannel)),
		PaymentReliabilityScore: &h.PaymentReliabilityScore,
		DunningRiskTier:         nilIfEmpty(h.DunningRiskTier),
		FirstDunningAt:          nullTime(h.FirstDunningAt),
		LastDunningAt:           nullTime(h.LastDunningAt),
		LastRecoveryAt:          nullTime(h.LastRecoveryAt),
		UpdatedAt:               h.UpdatedAt,
	}
}
