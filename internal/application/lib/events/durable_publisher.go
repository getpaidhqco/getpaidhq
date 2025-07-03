package events

import (
	"context"
)

// DurableEventPublisher is the interface for publishing durable events to Kafka
// These events are used for audit trails, compliance, and downstream processing
type DurableEventPublisher interface {
	// Core Business Events - Audit Trail & Compliance
	PublishUsageEvent(ctx context.Context, event UsageRecordedEvent) error
	PublishBillingEvent(ctx context.Context, event BillingEvent) error
	PublishPaymentEvent(ctx context.Context, event PaymentEvent) error
	PublishSubscriptionEvent(ctx context.Context, event SubscriptionEvent) error
	PublishCustomerEvent(ctx context.Context, event CustomerEvent) error
	PublishInvoiceEvent(ctx context.Context, event InvoiceEvent) error
	PublishRefundEvent(ctx context.Context, event RefundEvent) error
	PublishProductEvent(ctx context.Context, event ProductEvent) error
	PublishPriceEvent(ctx context.Context, event PriceEvent) error
	PublishDunningEvent(ctx context.Context, event DunningEvent) error
	
	// Batch operations for high-volume events
	PublishUsageBatch(ctx context.Context, events []UsageRecordedEvent) error
	PublishEventBatch(ctx context.Context, events []BaseEvent) error
}

// Event type constants for Kafka events
const (
	// Usage Events
	UsageRecorded      = "usage.recorded"
	UsageBatchRecorded = "usage.batch.recorded"
	UsageAggregated    = "usage.aggregated"
	
	// Billing Events
	BillingInvoiceCreated   = "billing.invoice.created"
	BillingInvoicePaid      = "billing.invoice.paid"
	BillingInvoiceOverdue   = "billing.invoice.overdue"
	BillingAmountCalculated = "billing.amount.calculated"
	
	// Subscription Events
	SubscriptionCreated     = "subscription.created"
	SubscriptionActivated   = "subscription.activated"
	SubscriptionPaused      = "subscription.paused"
	SubscriptionResumed     = "subscription.resumed"
	SubscriptionCancelled   = "subscription.cancelled"
	SubscriptionExpired     = "subscription.expired"
	SubscriptionPlanChanged = "subscription.plan.changed"
	
	// Payment Events
	PaymentCreated   = "payment.created"
	PaymentSucceeded = "payment.succeeded"
	PaymentFailed    = "payment.failed"
	PaymentRefunded  = "payment.refunded"
	
	// Customer Events
	CustomerCreated             = "customer.created"
	CustomerUpdated             = "customer.updated"
	CustomerPaymentMethodUpdated = "customer.payment_method.updated"
	CustomerDeleted             = "customer.deleted"
	
	// Product Events
	ProductCreated = "product.created"
	ProductUpdated = "product.updated"
	ProductDeleted = "product.deleted"
	PriceCreated   = "price.created"
	PriceUpdated   = "price.updated"
	PriceDeleted   = "price.deleted"
	
	// Dunning Events
	DunningCampaignStarted        = "dunning.campaign.started"
	DunningPaymentRecovered       = "dunning.payment.recovered"
	DunningCampaignFailed         = "dunning.campaign.failed"
	DunningSubscriptionSuspended  = "dunning.subscription.suspended"
	
	// Audit Events
	AuditUserAction    = "audit.user.action"
	AuditSystemAction  = "audit.system.action"
	AuditDataChange    = "audit.data.change"
	AuditSecurityEvent = "audit.security.event"
)

// Kafka topic names
const (
	TopicUsageEvents    = "gphq.usage.recorded"
	TopicBillingEvents  = "gphq.billing.events"
	TopicPaymentEvents  = "gphq.payment.events"
	TopicCustomerEvents = "gphq.customer.events"
	TopicAuditEvents    = "gphq.audit.events"
)