package port

import (
	"getpaidhq/internal/core/domain"
	"time"
)

// Event topic constants for pub/sub messaging.
const (
	TopicOrgCreated = "org.created"

	TopicPaymentChargeSuccess = "payment.charge.success"
	TopicChargeFailed         = "charge.failed"
	TopicProductCreated       = "product.created"
	TopicProductUpdated       = "product.updated"
	TopicProductDeleted       = "product.deleted"
	TopicProductArchived      = "product.archived"
	TopicProductUnarchived    = "product.unarchived"

	TopicVariantCreated = "variant.created"
	TopicVariantUpdated = "variant.updated"
	TopicVariantDeleted = "variant.deleted"

	TopicPriceCreated = "price.created"
	TopicPriceUpdated = "price.updated"
	TopicPriceDeleted = "price.deleted"

	TopicOrderCompleted = "order.completed"

	TopicCustomerCreated = "customer.created"

	TopicSubscriptionCreated              = "subscription.created"
	TopicSubscriptionPaused               = "subscription.paused"
	TopicSubscriptionActivated            = "subscription.activated"
	TopicSubscriptionResumed              = "subscription.resumed"
	TopicSubscriptionCancelled            = "subscription.cancelled"
	TopicSubscriptionUnpaid               = "subscription.unpaid"
	TopicSubscriptionExpired              = "subscription.expired"
	TopicSubscriptionCompleted            = "subscription.completed"
	TopicSubscriptionPastDue              = "subscription.past_due"
	TopicSubscriptionRenewalReminder      = "subscription.renewal_reminder"
	TopicSubscriptionBillingAnchorChanged = "subscription.billing_anchor_changed"

	TopicPaymentCreated = "payment.created"
	TopicPaymentUpdated = "payment.updated"
	TopicPaymentDeleted = "payment.deleted"
	TopicPaymentFailed  = "payment.failed"

	TopicPaymentMethodCreated   = "payment_method.created"
	TopicPaymentMethodUpdated   = "payment_method.updated"
	TopicPaymentMethodDeleted   = "payment_method.deleted"
	TopicPaymentMethodExpired   = "payment_method.expired"
	TopicPaymentMethodExpiryDue = "payment_method.expiry_due"

	TopicSubscriptionPaymentChargeSuccess = "subscription.payment.charge.success"
	TopicSubscriptionPaymentChargeFailed  = "subscription.payment.charge.failed"

	TopicSubscriptionWorkflowStartupFailed = "subscription.workflow.startup.failed"

	TopicWebhookSubscriptionCreated = "webhook.created"

	TopicSessionCreated = "session.created"

	// Dunning lifecycle topics. The orchestrator subscribes to
	// subscription.payment.charge.failed and starts a campaign; downstream
	// services react to these topics to send notifications / update history.
	TopicDunningCampaignStarted         = "dunning.started"
	TopicDunningCampaignPaused          = "dunning.paused"
	TopicDunningCampaignResumed         = "dunning.resumed"
	TopicDunningCampaignCancelled       = "dunning.cancelled"
	TopicDunningCampaignRecovered       = "dunning.recovered"
	TopicDunningCampaignFailed          = "dunning.failed"
	TopicDunningCampaignExpired         = "dunning.expired"
	TopicDunningAttemptFailed           = "dunning.attempt_failed"
	TopicDunningAttemptSucceeded        = "dunning.attempt_succeeded"
	TopicDunningPaymentRecovered        = "dunning.payment_recovered"
	TopicDunningFinalFailure            = "dunning.final_failure"
	TopicDunningSubscriptionSuspended   = "dunning.subscription_suspended"
	TopicDunningSubscriptionReactivated = "dunning.subscription_reactivated"
	TopicDunningPaymentMethodUpdated    = "dunning.payment_method_updated"
	TopicDunningTokenCreated            = "dunning.token_created"
	TopicDunningTokenActivated          = "dunning.token_activated"
	TopicDunningTokenRevoked            = "dunning.token_revoked"
	TopicDunningTokenExpired            = "dunning.token_expired"
	TopicDunningCommunicationSent       = "dunning.communication_sent"
	TopicDunningCommunicationFailed     = "dunning.communication_failed"
	TopicDunningConfigurationCreated    = "dunning.configuration_created"
	TopicDunningConfigurationUpdated    = "dunning.configuration_updated"
)

// GetSubscriptionTopic returns the pub/sub topic for a given subscription status.
func GetSubscriptionTopic(status domain.SubscriptionStatus) string {
	switch status {
	case domain.SubscriptionStatusActive:
		return TopicSubscriptionActivated
	case domain.SubscriptionStatusPaused:
		return TopicSubscriptionPaused
	case domain.SubscriptionStatusCancelled:
		return TopicSubscriptionCancelled
	case domain.SubscriptionStatusExpired:
		return TopicSubscriptionExpired
	case domain.SubscriptionStatusCompleted:
		return TopicSubscriptionCompleted
	case domain.SubscriptionStatusPastDue:
		return TopicSubscriptionPastDue
	case domain.SubscriptionStatusUnpaid:
		return TopicSubscriptionUnpaid
	default:
		return ""
	}
}

// SubscriptionPaymentChargeSuccessEvent is published when a subscription charge succeeds.
type SubscriptionPaymentChargeSuccessEvent struct {
	OrgId          string            `json:"org_id"`
	SubscriptionId string            `json:"subscription_id"`
	OrderId        string            `json:"order_id"`
	PaymentId      string            `json:"payment_id"`
	Metadata       map[string]string `json:"metadata"`
	Payment        domain.Payment    `json:"payment"`
}

func NewSubscriptionPaymentChargeSuccessEvent(sub domain.Subscription, payment domain.Payment) SubscriptionPaymentChargeSuccessEvent {
	return SubscriptionPaymentChargeSuccessEvent{
		OrgId:          sub.OrgId,
		SubscriptionId: sub.Id,
		OrderId:        sub.OrderId,
		PaymentId:      payment.Id,
		Metadata:       sub.Metadata,
		Payment:        payment,
	}
}

// ProrationDetails contains the calculated proration information for billing anchor changes.
type ProrationDetails struct {
	CreditAmount       int       `json:"credit_amount"`
	DaysCredited       int       `json:"days_credited"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	OldBillingAnchor   int       `json:"old_billing_anchor,omitempty"`
	NewBillingAnchor   int       `json:"new_billing_anchor,omitempty"`
	NewPeriodStart     time.Time `json:"new_period_start"`
	NewPeriodEnd       time.Time `json:"new_period_end"`
}

// SubscriptionBillingAnchorChangedData contains the event-specific data for billing anchor changes.
type SubscriptionBillingAnchorChangedData struct {
	OldBillingAnchor int               `json:"old_billing_anchor"`
	NewBillingAnchor int               `json:"new_billing_anchor"`
	ProrationMode    string            `json:"proration_mode"`
	ProrationDetails *ProrationDetails `json:"proration_details,omitempty"`
	EffectiveDate    time.Time         `json:"effective_date"`
}

// SubscriptionBillingAnchorChangedEvent represents the complete billing anchor changed event.
type SubscriptionBillingAnchorChangedEvent struct {
	Event          string                               `json:"event"`
	SubscriptionID string                               `json:"subscription_id"`
	OrgID          string                               `json:"org_id"`
	Data           SubscriptionBillingAnchorChangedData `json:"data"`
	CreatedAt      time.Time                            `json:"created_at"`
}

// NewSubscriptionBillingAnchorChangedEvent creates a new billing anchor changed event.
func NewSubscriptionBillingAnchorChangedEvent(
	subscriptionID,
	orgID string,
	oldAnchor,
	newAnchor int,
	prorationMode string,
	prorationDetails *ProrationDetails,
) *SubscriptionBillingAnchorChangedEvent {
	return &SubscriptionBillingAnchorChangedEvent{
		Event:          "subscription.billing_anchor_changed",
		SubscriptionID: subscriptionID,
		OrgID:          orgID,
		Data: SubscriptionBillingAnchorChangedData{
			OldBillingAnchor: oldAnchor,
			NewBillingAnchor: newAnchor,
			ProrationMode:    prorationMode,
			ProrationDetails: prorationDetails,
			EffectiveDate:    time.Now(),
		},
		CreatedAt: time.Now(),
	}
}
