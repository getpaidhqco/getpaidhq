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
	OrgId string
	Id    string

	SubscriptionId string
	CustomerId     string

	// WorkflowId / WorkflowRunId identify the running engine workflow handle.
	WorkflowId       string
	WorkflowRunId    string
	ParentWorkflowId string

	Status               DunningStatus
	FailedAmount         int64
	Currency             string
	InitialFailureReason string

	TotalAttempts       int
	ImmediateAttempts   int
	ProgressiveAttempts int

	StartedAt     time.Time
	LastAttemptAt time.Time
	NextAttemptAt time.Time
	CompletedAt   time.Time

	RecoveryMethod     string
	RecoveredAmount    int64
	RecoveredAt        time.Time
	FinalFailureReason string

	ConfigSnapshot map[string]any
	Metadata       map[string]string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// DunningAttempt records a single charge attempt within a campaign.
type DunningAttempt struct {
	OrgId string
	Id    string

	DunningCampaignId string
	SubscriptionId    string

	AttemptNumber int
	AttemptType   DunningAttemptType

	Amount          int64
	Currency        string
	PaymentMethodId string

	Status            PaymentStatus
	FailureReason     string
	FailureCode       string
	ProcessorResponse map[string]any

	ProcessingTimeMs int
	AttemptedAt      time.Time
	CompletedAt      time.Time

	TriggeredBy string
	Metadata    map[string]string

	CreatedAt time.Time
}

// DunningCommunication records an outbound message sent to a customer as part
// of a dunning campaign.
type DunningCommunication struct {
	OrgId string
	Id    string

	DunningCampaignId string
	CustomerId        string

	Channel       CommunicationChannel
	TemplateId    string
	AttemptNumber int

	Subject             string
	ContentPreview      string
	PersonalizationData map[string]any

	SentAt      time.Time
	DeliveredAt time.Time
	OpenedAt    time.Time
	ClickedAt   time.Time
	BouncedAt   time.Time

	Provider          string
	ProviderMessageId string
	ProviderResponse  map[string]any

	Status        CommunicationStatus
	FailureReason string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// PaymentUpdateToken is a one-or-few-use signed link delivered to customers
// during dunning so they can update their payment method without logging in.
type PaymentUpdateToken struct {
	OrgId   string
	TokenId string

	SubscriptionId    string
	CustomerId        string
	DunningCampaignId string

	TokenData map[string]any
	Signature string

	ExpiresAt time.Time
	MaxUses   int
	UsedCount int

	Status         TokenStatus
	AllowedActions map[string]bool

	AdminGenerated bool
	AdminUserId    string
	AdminReason    string
	AdminNotes     string

	CreatedBy  string
	CreatedAt  time.Time
	LastUsedAt time.Time
	LastUsedIp string
}

// DunningConfiguration is a named, scoped retry-policy + communication-policy
// bundle. Multiple configs can be defined; the orchestrator picks the
// highest-priority match for a given subscription.
type DunningConfiguration struct {
	OrgId string
	Id    string

	Name        string
	Description string
	Priority    int

	AppliesTo   DunningConfigScope
	TargetRules map[string]any
	Config      map[string]any

	Status           ConfigStatus
	IsAbTest         bool
	AbTestPercentage float64

	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CustomerDunningHistory is the per-customer rolled-up dunning record (filled
// in over time by the orchestrator as campaigns close out).
type CustomerDunningHistory struct {
	OrgId      string
	CustomerId string

	TotalDunningCampaigns int
	SuccessfulRecoveries  int
	FailedCampaigns       int

	TotalAmountAtRisk    int64
	TotalAmountRecovered int64
	TotalAmountLost      int64

	AvgRecoveryTimeHours    float64
	PreferredRecoveryMethod string
	MostResponsiveChannel   CommunicationChannel
	PaymentReliabilityScore float64
	DunningRiskTier         string

	FirstDunningAt time.Time
	LastDunningAt  time.Time
	LastRecoveryAt time.Time

	UpdatedAt time.Time
}
