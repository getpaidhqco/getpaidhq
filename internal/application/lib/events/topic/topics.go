package topic

const (
	// Org
	OrgCreated = "org.created"

	TopicPaymentChargeSuccess = "payment.charge.success"
	TopicChargeFailed         = "charge.failed"
	ProductCreated            = "product.created"
	ProductUpdated            = "product.updated"
	ProductDeleted            = "product.deleted"

	VariantCreated            = "variant.created"
	VariantUpdated            = "variant.updated"
	VariantDeleted            = "variant.deleted"

	PriceCreated              = "price.created"
	PriceUpdated              = "price.updated"
	PriceDeleted              = "price.deleted"

	OrderCompleted = "order.completed"

	CustomerCreated = "customer.created"

	TopicSubscriptionCreated         = "subscription.created"
	TopicSubscriptionPaused          = "subscription.paused"
	TopicSubscriptionActivated       = "subscription.activated"
	TopicSubscriptionResumed         = "subscription.resumed"
	TopicSubscriptionCancelled       = "subscription.cancelled"
	TopicSubscriptionUnpaid          = "subscription.unpaid"
	SubscriptionStatusExpired        = "subscription.expired"
	SubscriptionStatusCompleted      = "subscription.completed"
	SubscriptionStatusPastDue        = "subscription.past_due"
	SubscriptionRenewalReminder      = "subscription.renewal_reminder"
	SubscriptionBillingAnchorChanged = "subscription.billing_anchor_changed"

	PaymentCreated = "payment.created"
	PaymentUpdated = "payment.updated"
	PaymentDeleted = "payment.deleted"
	PaymentFailed  = "payment.failed"

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
