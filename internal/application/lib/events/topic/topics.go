package topic

const (
	TopicPaymentChargeSuccess = "payment.charge.success"
	TopicChargeFailed         = "charge.failed"
	TopicTransferSuccess      = "transfer.success"
	TopicTransferFailed       = "transfer.failed"

	OrderCompleted = "order.completed"

	TopicSubscriptionCreated   = "subscription.created"
	TopicSubscriptionPaused    = "subscription.paused"
	TopicSubscriptionActivated = "subscription.activated"
	TopicSubscriptionResumed   = "subscription.resumed"
	TopicSubscriptionCancelled = "subscription.cancelled"
	SubscriptionStatusExpired  = "subscription.expired"
	SubscriptionStatusPastDue  = "subscription.past_due"

	SubscriptionPaymentChargeSuccess = "subscription.payment.charge.success"
	SubscriptionPaymentChargeFailed  = "subscription.payment.charge.failed"

	SubscriptionWorkflowStartupFailed = "subscription.workflow.startup.failed"

	WebhookSubscriptionCreated = "webhook.created"

	SessionCreated = "session.created"
)
