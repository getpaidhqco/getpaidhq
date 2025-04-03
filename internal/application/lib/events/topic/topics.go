package topic

const (
	TopicPaymentChargeSuccess = "payment.charge.success"
	TopicChargeFailed         = "charge.failed"
	ProductCreated            = "product.created"

	PriceCreated = "price.created"

	OrderCompleted = "order.completed"

	TopicSubscriptionCreated    = "subscription.created"
	TopicSubscriptionPaused     = "subscription.paused"
	TopicSubscriptionActivated  = "subscription.activated"
	TopicSubscriptionResumed    = "subscription.resumed"
	TopicSubscriptionCancelled  = "subscription.cancelled"
	TopicSubscriptionUnpaid     = "subscription.unpaid"
	SubscriptionStatusExpired   = "subscription.expired"
	SubscriptionStatusCompleted = "subscription.completed"
	SubscriptionStatusPastDue   = "subscription.past_due"
	SubscriptionRenewalReminder = "subscription.renewal_reminder"

	// Payment Method
	PaymentMethodCreated   = "payment_method.created"
	PaymentMethodUpdated   = "payment_method.updated"
	PaymentMethodDeleted   = "payment_method.deleted"
	PaymentMethodExpired   = "payment_method.expired"
	PaymentMethodExpiryDue = "payment_method.expiry_due"

	SubscriptionPaymentChargeSuccess = "subscription.payment.charge.success"
	SubscriptionPaymentChargeFailed  = "subscription.payment.charge.failed"

	SubscriptionWorkflowStartupFailed = "subscription.workflow.startup.failed"

	WebhookSubscriptionCreated = "webhook.created"

	SessionCreated = "session.created"
)
