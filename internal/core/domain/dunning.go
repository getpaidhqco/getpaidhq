package domain

import "time"

// DunningStatus is the lifecycle state of a dunning campaign.
type DunningStatus string

const (
	DunningStatusActive    DunningStatus = "active"
	DunningStatusPaused    DunningStatus = "paused"
	DunningStatusRecovered DunningStatus = "recovered"
	DunningStatusFailed    DunningStatus = "failed"
	DunningStatusCancelled DunningStatus = "cancelled"
	DunningStatusExpired   DunningStatus = "expired"
)

// DunningAttemptType differentiates immediate (transient-failure) retries
// from progressive (customer-driven) retries, plus admin/manual triggers.
type DunningAttemptType string

const (
	DunningAttemptTypeImmediate   DunningAttemptType = "immediate"
	DunningAttemptTypeProgressive DunningAttemptType = "progressive"
	DunningAttemptTypeManual      DunningAttemptType = "manual"
	DunningAttemptTypeTriggered   DunningAttemptType = "triggered"
)

type CommunicationChannel string

const (
	CommunicationChannelEmail   CommunicationChannel = "email"
	CommunicationChannelSMS     CommunicationChannel = "sms"
	CommunicationChannelPush    CommunicationChannel = "push"
	CommunicationChannelWebhook CommunicationChannel = "webhook"
	CommunicationChannelInApp   CommunicationChannel = "in_app"
)

type CommunicationStatus string

const (
	CommunicationStatusPending   CommunicationStatus = "pending"
	CommunicationStatusSent      CommunicationStatus = "sent"
	CommunicationStatusDelivered CommunicationStatus = "delivered"
	CommunicationStatusFailed    CommunicationStatus = "failed"
	CommunicationStatusBounced   CommunicationStatus = "bounced"
)

type TokenStatus string

const (
	TokenStatusActive         TokenStatus = "active"
	TokenStatusExpired        TokenStatus = "expired"
	TokenStatusRevoked        TokenStatus = "revoked"
	TokenStatusMaxUsesReached TokenStatus = "max_uses_reached"
)

type DunningConfigScope string

const (
	DunningConfigScopeOrganization     DunningConfigScope = "organization"
	DunningConfigScopeCustomerSegment  DunningConfigScope = "customer_segment"
	DunningConfigScopeSubscriptionTier DunningConfigScope = "subscription_tier"
	DunningConfigScopeCustomer         DunningConfigScope = "customer"
	DunningConfigScopeAbTest           DunningConfigScope = "ab_test"
)

type ConfigStatus string

const (
	ConfigStatusActive   ConfigStatus = "active"
	ConfigStatusInactive ConfigStatus = "inactive"
	ConfigStatusArchived ConfigStatus = "archived"
)

// DunningCampaign is a single recovery campaign for a failed subscription
// charge. One campaign per failed charge; created by the orchestrator when a
// charge fails and torn down when recovered/cancelled/exhausted.
type DunningCampaign struct {
	OrgId string `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id    string `gorm:"column:id;primaryKey" json:"id"`

	SubscriptionId string `gorm:"column:subscription_id" json:"subscription_id"`
	CustomerId     string `gorm:"column:customer_id" json:"customer_id"`

	// WorkflowId / WorkflowRunId identify the running engine workflow handle.
	WorkflowId       string `gorm:"column:workflow_id" json:"workflow_id"`
	WorkflowRunId    string `gorm:"column:workflow_run_id" json:"workflow_run_id"`
	ParentWorkflowId string `gorm:"column:parent_workflow_id" json:"parent_workflow_id,omitempty"`

	Status               DunningStatus `gorm:"column:status" json:"status"`
	FailedAmount         int64         `gorm:"column:failed_amount" json:"failed_amount"`
	Currency             string        `gorm:"column:currency" json:"currency"`
	InitialFailureReason string        `gorm:"column:initial_failure_reason" json:"initial_failure_reason,omitempty"`

	TotalAttempts       int `gorm:"column:total_attempts" json:"total_attempts"`
	ImmediateAttempts   int `gorm:"column:immediate_attempts" json:"immediate_attempts"`
	ProgressiveAttempts int `gorm:"column:progressive_attempts" json:"progressive_attempts"`

	StartedAt     time.Time `gorm:"column:started_at" json:"started_at"`
	LastAttemptAt time.Time `gorm:"column:last_attempt_at" json:"last_attempt_at,omitzero"`
	NextAttemptAt time.Time `gorm:"column:next_attempt_at" json:"next_attempt_at,omitzero"`
	CompletedAt   time.Time `gorm:"column:completed_at" json:"completed_at,omitzero"`

	RecoveryMethod     string    `gorm:"column:recovery_method" json:"recovery_method,omitempty"`
	RecoveredAmount    int64     `gorm:"column:recovered_amount" json:"recovered_amount,omitempty"`
	RecoveredAt        time.Time `gorm:"column:recovered_at" json:"recovered_at,omitzero"`
	FinalFailureReason string    `gorm:"column:final_failure_reason" json:"final_failure_reason,omitempty"`

	ConfigSnapshot map[string]any    `gorm:"column:config_snapshot;serializer:json" json:"config_snapshot,omitempty"`
	Metadata       map[string]string `gorm:"column:metadata;serializer:json" json:"metadata,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (DunningCampaign) TableName() string { return "dunning_campaigns" }

// DunningAttempt records a single charge attempt within a campaign.
type DunningAttempt struct {
	OrgId string `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id    string `gorm:"column:id;primaryKey" json:"id"`

	DunningCampaignId string `gorm:"column:dunning_campaign_id" json:"dunning_campaign_id"`
	SubscriptionId    string `gorm:"column:subscription_id" json:"subscription_id"`

	AttemptNumber int                `gorm:"column:attempt_number" json:"attempt_number"`
	AttemptType   DunningAttemptType `gorm:"column:attempt_type" json:"attempt_type"`

	Amount          int64  `gorm:"column:amount" json:"amount"`
	Currency        string `gorm:"column:currency" json:"currency"`
	PaymentMethodId string `gorm:"column:payment_method_id" json:"payment_method_id,omitempty"`

	Status            PaymentStatus  `gorm:"column:status" json:"status"`
	FailureReason     string         `gorm:"column:failure_reason" json:"failure_reason,omitempty"`
	FailureCode       string         `gorm:"column:failure_code" json:"failure_code,omitempty"`
	ProcessorResponse map[string]any `gorm:"column:processor_response;serializer:json" json:"processor_response,omitempty"`

	ProcessingTimeMs int       `gorm:"column:processing_time_ms" json:"processing_time_ms,omitempty"`
	AttemptedAt      time.Time `gorm:"column:attempted_at" json:"attempted_at"`
	CompletedAt      time.Time `gorm:"column:completed_at" json:"completed_at,omitzero"`

	TriggeredBy string            `gorm:"column:triggered_by" json:"triggered_by,omitempty"`
	Metadata    map[string]string `gorm:"column:metadata;serializer:json" json:"metadata,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (DunningAttempt) TableName() string { return "dunning_attempts" }

// DunningCommunication records an outbound message sent to a customer as part
// of a dunning campaign.
type DunningCommunication struct {
	OrgId string `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id    string `gorm:"column:id;primaryKey" json:"id"`

	DunningCampaignId string `gorm:"column:dunning_campaign_id" json:"dunning_campaign_id"`
	CustomerId        string `gorm:"column:customer_id" json:"customer_id"`

	Channel       CommunicationChannel `gorm:"column:channel" json:"channel"`
	TemplateId    string               `gorm:"column:template_id" json:"template_id"`
	AttemptNumber int                  `gorm:"column:attempt_number" json:"attempt_number"`

	Subject             string         `gorm:"column:subject" json:"subject,omitempty"`
	ContentPreview      string         `gorm:"column:content_preview" json:"content_preview,omitempty"`
	PersonalizationData map[string]any `gorm:"column:personalization_data;serializer:json" json:"personalization_data,omitempty"`

	SentAt      time.Time `gorm:"column:sent_at" json:"sent_at,omitzero"`
	DeliveredAt time.Time `gorm:"column:delivered_at" json:"delivered_at,omitzero"`
	OpenedAt    time.Time `gorm:"column:opened_at" json:"opened_at,omitzero"`
	ClickedAt   time.Time `gorm:"column:clicked_at" json:"clicked_at,omitzero"`
	BouncedAt   time.Time `gorm:"column:bounced_at" json:"bounced_at,omitzero"`

	Provider          string         `gorm:"column:provider" json:"provider"`
	ProviderMessageId string         `gorm:"column:provider_message_id" json:"provider_message_id,omitempty"`
	ProviderResponse  map[string]any `gorm:"column:provider_response;serializer:json" json:"provider_response,omitempty"`

	Status        CommunicationStatus `gorm:"column:status" json:"status"`
	FailureReason string              `gorm:"column:failure_reason" json:"failure_reason,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (DunningCommunication) TableName() string { return "dunning_communications" }

// PaymentUpdateToken is a one-or-few-use signed link delivered to customers
// during dunning so they can update their payment method without logging in.
type PaymentUpdateToken struct {
	OrgId   string `gorm:"column:org_id;primaryKey" json:"org_id"`
	TokenId string `gorm:"column:token_id;primaryKey" json:"token_id"`

	SubscriptionId    string `gorm:"column:subscription_id" json:"subscription_id"`
	CustomerId        string `gorm:"column:customer_id" json:"customer_id"`
	DunningCampaignId string `gorm:"column:dunning_campaign_id" json:"dunning_campaign_id,omitempty"`

	TokenData map[string]any `gorm:"column:token_data;serializer:json" json:"token_data,omitempty"`
	Signature string         `gorm:"column:signature" json:"signature"`

	ExpiresAt time.Time `gorm:"column:expires_at" json:"expires_at"`
	MaxUses   int       `gorm:"column:max_uses" json:"max_uses"`
	UsedCount int       `gorm:"column:used_count" json:"used_count"`

	Status         TokenStatus     `gorm:"column:status" json:"status"`
	AllowedActions map[string]bool `gorm:"column:allowed_actions;serializer:json" json:"allowed_actions,omitempty"`

	AdminGenerated bool   `gorm:"column:admin_generated" json:"admin_generated"`
	AdminUserId    string `gorm:"column:admin_user_id" json:"admin_user_id,omitempty"`
	AdminReason    string `gorm:"column:admin_reason" json:"admin_reason,omitempty"`
	AdminNotes     string `gorm:"column:admin_notes" json:"admin_notes,omitempty"`

	CreatedBy  string    `gorm:"column:created_by" json:"created_by,omitempty"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	LastUsedAt time.Time `gorm:"column:last_used_at" json:"last_used_at,omitzero"`
	LastUsedIp string    `gorm:"column:last_used_ip" json:"last_used_ip,omitempty"`
}

func (PaymentUpdateToken) TableName() string { return "payment_update_tokens" }

// DunningConfiguration is a named, scoped retry-policy + communication-policy
// bundle. Multiple configs can be defined; the orchestrator picks the
// highest-priority match for a given subscription.
type DunningConfiguration struct {
	OrgId string `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id    string `gorm:"column:id;primaryKey" json:"id"`

	Name        string `gorm:"column:name" json:"name"`
	Description string `gorm:"column:description" json:"description,omitempty"`
	Priority    int    `gorm:"column:priority" json:"priority"`

	AppliesTo   DunningConfigScope `gorm:"column:applies_to" json:"applies_to"`
	TargetRules map[string]any     `gorm:"column:target_rules;serializer:json" json:"target_rules,omitempty"`
	Config      map[string]any     `gorm:"column:config;serializer:json" json:"config"`

	Status           ConfigStatus `gorm:"column:status" json:"status"`
	IsAbTest         bool         `gorm:"column:is_ab_test" json:"is_ab_test"`
	AbTestPercentage float64      `gorm:"column:ab_test_percentage" json:"ab_test_percentage,omitempty"`

	CreatedBy string    `gorm:"column:created_by" json:"created_by,omitempty"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (DunningConfiguration) TableName() string { return "dunning_configurations" }

// CustomerDunningHistory is the per-customer rolled-up dunning record (filled
// in over time by the orchestrator as campaigns close out).
type CustomerDunningHistory struct {
	OrgId      string `gorm:"column:org_id;primaryKey" json:"org_id"`
	CustomerId string `gorm:"column:customer_id;primaryKey" json:"customer_id"`

	TotalDunningCampaigns int `gorm:"column:total_dunning_campaigns" json:"total_dunning_campaigns"`
	SuccessfulRecoveries  int `gorm:"column:successful_recoveries" json:"successful_recoveries"`
	FailedCampaigns       int `gorm:"column:failed_campaigns" json:"failed_campaigns"`

	TotalAmountAtRisk    int64 `gorm:"column:total_amount_at_risk" json:"total_amount_at_risk"`
	TotalAmountRecovered int64 `gorm:"column:total_amount_recovered" json:"total_amount_recovered"`
	TotalAmountLost      int64 `gorm:"column:total_amount_lost" json:"total_amount_lost"`

	AvgRecoveryTimeHours    float64              `gorm:"column:avg_recovery_time_hours" json:"avg_recovery_time_hours,omitempty"`
	PreferredRecoveryMethod string               `gorm:"column:preferred_recovery_method" json:"preferred_recovery_method,omitempty"`
	MostResponsiveChannel   CommunicationChannel `gorm:"column:most_responsive_channel" json:"most_responsive_channel,omitempty"`
	PaymentReliabilityScore float64              `gorm:"column:payment_reliability_score" json:"payment_reliability_score,omitempty"`
	DunningRiskTier         string               `gorm:"column:dunning_risk_tier" json:"dunning_risk_tier,omitempty"`

	FirstDunningAt time.Time `gorm:"column:first_dunning_at" json:"first_dunning_at,omitzero"`
	LastDunningAt  time.Time `gorm:"column:last_dunning_at" json:"last_dunning_at,omitzero"`
	LastRecoveryAt time.Time `gorm:"column:last_recovery_at" json:"last_recovery_at,omitzero"`

	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (CustomerDunningHistory) TableName() string { return "customer_dunning_history" }
