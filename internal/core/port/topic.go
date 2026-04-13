package port

import "payloop/internal/core/domain"

// Event topic constants for pub/sub messaging.
const (
	TopicOrgCreated = "org.created"

	TopicPaymentChargeSuccess = "payment.charge.success"
	TopicChargeFailed         = "charge.failed"
	TopicProductCreated       = "product.created"
	TopicProductUpdated       = "product.updated"
	TopicProductDeleted       = "product.deleted"

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
	Subscription domain.Subscription `json:"subscription"`
	Payment      domain.Payment      `json:"payment"`
}

func NewSubscriptionPaymentChargeSuccessEvent(sub domain.Subscription, pmt domain.Payment) SubscriptionPaymentChargeSuccessEvent {
	return SubscriptionPaymentChargeSuccessEvent{
		Subscription: sub,
		Payment:      pmt,
	}
}
