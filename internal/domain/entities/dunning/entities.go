package dunning

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"time"
)

// DunningCampaign represents a dunning campaign for a subscription
type DunningCampaign struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"`

	// Relationships
	SubscriptionId string                 `json:"subscription_id"`
	Subscription   *entities.Subscription `json:"-"`
	CustomerId     string                 `json:"customer_id"`
	Customer       *entities.Customer     `json:"-"`

	// Workflow metadata
	TemporalWorkflowId string `json:"temporal_workflow_id"`
	TemporalRunId      string `json:"temporal_run_id"`
	ParentWorkflowId   string `json:"parent_workflow_id,omitempty"`

	// Campaign details
	Status               DunningStatus `json:"status"`
	FailedAmount         int           `json:"failed_amount"`
	Currency             string        `json:"currency"`
	InitialFailureReason string        `json:"initial_failure_reason,omitempty"`

	// Attempt tracking
	TotalAttempts       int `json:"total_attempts"`
	ImmediateAttempts   int `json:"immediate_attempts"`
	ProgressiveAttempts int `json:"progressive_attempts"`

	// Timeline
	StartedAt     time.Time `json:"started_at"`
	LastAttemptAt time.Time `json:"last_attempt_at,omitempty"`
	NextAttemptAt time.Time `json:"next_attempt_at,omitempty"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`

	// Outcomes
	RecoveryMethod     string    `json:"recovery_method,omitempty"`
	RecoveredAmount    int       `json:"recovered_amount,omitempty"`
	RecoveredAt        time.Time `json:"recovered_at,omitempty"`
	FinalFailureReason string    `json:"final_failure_reason,omitempty"`

	// Configuration snapshot
	ConfigSnapshot map[string]interface{} `json:"config_snapshot,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DunningAttempt represents a payment retry attempt in a dunning campaign
type DunningAttempt struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"`

	// Relationships
	DunningCampaignId string           `json:"dunning_campaign_id"`
	Campaign          *DunningCampaign `json:"-"`
	SubscriptionId    string           `json:"subscription_id"`

	// Attempt details
	AttemptNumber int                `json:"attempt_number"`
	AttemptType   DunningAttemptType `json:"attempt_type"`

	// Payment details
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	PaymentMethodId string `json:"payment_method_id,omitempty"`

	// Results
	Status            payments.PaymentStatus `json:"status"`
	FailureReason     string                 `json:"failure_reason,omitempty"`
	FailureCode       string                 `json:"failure_code,omitempty"`
	ProcessorResponse string                 `json:"processor_response,omitempty"`

	// Performance metrics
	ProcessingTimeMs int       `json:"processing_time_ms,omitempty"`
	AttemptedAt      time.Time `json:"attempted_at"`
	CompletedAt      time.Time `json:"completed_at,omitempty"`

	// Context
	TriggeredBy string            `json:"triggered_by,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// DunningCommunication represents a communication sent to a customer during a dunning campaign
type DunningCommunication struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"`

	// Relationships
	DunningCampaignId string             `json:"dunning_campaign_id"`
	Campaign          *DunningCampaign   `json:"-"`
	CustomerId        string             `json:"customer_id"`
	Customer          *entities.Customer `json:"-"`

	// Communication details
	Channel       CommunicationChannel `json:"channel"`
	TemplateId    string               `json:"template_id"`
	AttemptNumber int                  `json:"attempt_number"`

	// Content
	Subject             string                 `json:"subject,omitempty"`
	ContentPreview      string                 `json:"content_preview,omitempty"`
	PersonalizationData map[string]interface{} `json:"personalization_data,omitempty"`

	// Delivery tracking
	SentAt      time.Time `json:"sent_at,omitempty"`
	DeliveredAt time.Time `json:"delivered_at,omitempty"`
	OpenedAt    time.Time `json:"opened_at,omitempty"`
	ClickedAt   time.Time `json:"clicked_at,omitempty"`
	BouncedAt   time.Time `json:"bounced_at,omitempty"`

	// Provider details
	Provider          string                 `json:"provider"`
	ProviderMessageId string                 `json:"provider_message_id,omitempty"`
	ProviderResponse  map[string]interface{} `json:"provider_response,omitempty"`

	// Status
	Status        CommunicationStatus `json:"status"`
	FailureReason string              `json:"failure_reason,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PaymentUpdateToken represents a secure token for updating payment methods
type PaymentUpdateToken struct {
	OrgId   string `json:"org_id"`
	TokenId string `json:"token_id"`

	// Relationships
	SubscriptionId    string                 `json:"subscription_id"`
	Subscription      *entities.Subscription `json:"-"`
	CustomerId        string                 `json:"customer_id"`
	Customer          *entities.Customer     `json:"-"`
	DunningCampaignId string                 `json:"dunning_campaign_id,omitempty"`
	Campaign          *DunningCampaign       `json:"-"`

	// Token data
	TokenData map[string]interface{} `json:"token_data"`
	Signature string                 `json:"signature"`

	// Security & usage
	ExpiresAt time.Time   `json:"expires_at"`
	MaxUses   int         `json:"max_uses"`
	UsedCount int         `json:"used_count"`
	Status    TokenStatus `json:"status"`

	// Allowed actions
	AllowedActions map[string]bool `json:"allowed_actions"`

	// Admin generation tracking
	AdminGenerated bool   `json:"admin_generated"`
	AdminUserId    string `json:"admin_user_id,omitempty"`
	AdminReason    string `json:"admin_reason,omitempty"`
	AdminNotes     string `json:"admin_notes,omitempty"`

	// Audit trail
	CreatedBy  string    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
	LastUsedIp string    `json:"last_used_ip,omitempty"`
}

// DunningConfiguration represents a configuration for dunning campaigns
type DunningConfiguration struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"`

	// Configuration hierarchy
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Priority    int    `json:"priority"`

	// Targeting rules
	AppliesTo   DunningConfigScope     `json:"applies_to"`
	TargetRules map[string]interface{} `json:"target_rules,omitempty"`

	// The actual configuration
	Config map[string]interface{} `json:"config"`

	// Status and testing
	Status           ConfigStatus `json:"status"`
	IsAbTest         bool         `json:"is_ab_test"`
	AbTestPercentage float64      `json:"ab_test_percentage,omitempty"`

	// Metadata
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CustomerDunningHistory represents a summary of a customer's dunning history
type CustomerDunningHistory struct {
	OrgId      string             `json:"org_id"`
	CustomerId string             `json:"customer_id"`
	Customer   *entities.Customer `json:"-"`

	// Lifetime stats
	TotalDunningCampaigns int `json:"total_dunning_campaigns"`
	SuccessfulRecoveries  int `json:"successful_recoveries"`
	FailedCampaigns       int `json:"failed_campaigns"`

	// Financial impact
	TotalAmountAtRisk    int `json:"total_amount_at_risk"`
	TotalAmountRecovered int `json:"total_amount_recovered"`
	TotalAmountLost      int `json:"total_amount_lost"`

	// Behavior patterns
	AvgRecoveryTimeHours    float64              `json:"avg_recovery_time_hours,omitempty"`
	PreferredRecoveryMethod string               `json:"preferred_recovery_method,omitempty"`
	MostResponsiveChannel   CommunicationChannel `json:"most_responsive_channel,omitempty"`

	// Risk scoring
	PaymentReliabilityScore float64 `json:"payment_reliability_score,omitempty"`
	DunningRiskTier         string  `json:"dunning_risk_tier,omitempty"`

	// Dates
	FirstDunningAt time.Time `json:"first_dunning_at,omitempty"`
	LastDunningAt  time.Time `json:"last_dunning_at,omitempty"`
	LastRecoveryAt time.Time `json:"last_recovery_at,omitempty"`

	UpdatedAt time.Time `json:"updated_at"`
}
