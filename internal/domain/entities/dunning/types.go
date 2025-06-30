package dunning

// DunningStatus represents the status of a dunning campaign
type DunningStatus string

const (
	DunningStatusActive    DunningStatus = "active"
	DunningStatusPaused    DunningStatus = "paused"
	DunningStatusRecovered DunningStatus = "recovered"
	DunningStatusFailed    DunningStatus = "failed"
	DunningStatusCancelled DunningStatus = "cancelled"
	DunningStatusExpired   DunningStatus = "expired"
)

// DunningAttemptType represents the type of a dunning attempt
type DunningAttemptType string

const (
	DunningAttemptTypeImmediate   DunningAttemptType = "immediate"
	DunningAttemptTypeProgressive DunningAttemptType = "progressive"
	DunningAttemptTypeManual      DunningAttemptType = "manual"
	DunningAttemptTypeTriggered   DunningAttemptType = "triggered"
)

// CommunicationChannel represents the channel used for communication
type CommunicationChannel string

const (
	CommunicationChannelEmail   CommunicationChannel = "email"
	CommunicationChannelSMS     CommunicationChannel = "sms"
	CommunicationChannelPush    CommunicationChannel = "push"
	CommunicationChannelWebhook CommunicationChannel = "webhook"
	CommunicationChannelInApp   CommunicationChannel = "in_app"
)

// CommunicationStatus represents the status of a communication
type CommunicationStatus string

const (
	CommunicationStatusPending   CommunicationStatus = "pending"
	CommunicationStatusSent      CommunicationStatus = "sent"
	CommunicationStatusDelivered CommunicationStatus = "delivered"
	CommunicationStatusFailed    CommunicationStatus = "failed"
	CommunicationStatusBounced   CommunicationStatus = "bounced"
)

// TokenStatus represents the status of a payment update token
type TokenStatus string

const (
	TokenStatusActive        TokenStatus = "active"
	TokenStatusExpired       TokenStatus = "expired"
	TokenStatusRevoked       TokenStatus = "revoked"
	TokenStatusMaxUsesReached TokenStatus = "max_uses_reached"
)

// DunningConfigScope represents the scope of a dunning configuration
type DunningConfigScope string

const (
	DunningConfigScopeOrganization     DunningConfigScope = "organization"
	DunningConfigScopeCustomerSegment  DunningConfigScope = "customer_segment"
	DunningConfigScopeSubscriptionTier DunningConfigScope = "subscription_tier"
	DunningConfigScopeCustomer         DunningConfigScope = "customer"
	DunningConfigScopeABTest           DunningConfigScope = "ab_test"
)

// ConfigStatus represents the status of a configuration
type ConfigStatus string

const (
	ConfigStatusActive   ConfigStatus = "active"
	ConfigStatusInactive ConfigStatus = "inactive"
	ConfigStatusArchived ConfigStatus = "archived"
)
