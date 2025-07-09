package repositories

import (
	"context"
	"time"

	"payloop/internal/domain/entities"
)

// UsageEventRepository defines the interface for usage event storage operations
type UsageEventRepository interface {
	// Create inserts a new usage event
	Create(ctx context.Context, event entities.UsageEvent) error

	// BatchCreate inserts multiple usage events efficiently
	BatchCreate(ctx context.Context, events []entities.UsageEvent) error

	// FindByID retrieves a usage event by composite key
	FindByID(ctx context.Context, orgID, subscriptionItemID string, time time.Time) (entities.UsageEvent, error)

	// FindBySubscriptionItem retrieves usage events for a subscription item
	FindBySubscriptionItem(ctx context.Context, orgID, subscriptionItemID string, 
		startTime, endTime time.Time) ([]entities.UsageEvent, error)

	// FindByReferenceID retrieves usage event by reference (for idempotency)
	FindByReferenceID(ctx context.Context, referenceID, referenceType string) (entities.UsageEvent, error)

	// Delete removes a usage event (for corrections)
	Delete(ctx context.Context, orgID, subscriptionItemID string, time time.Time) error

	// AggregateUsageBySubscriptionItem aggregates usage for a subscription item based on the specified aggregation type
	AggregateUsageBySubscriptionItem(ctx context.Context, orgID, subscriptionItemID string, 
		startTime, endTime time.Time, aggregationType entities.AggregationType) (float64, error)
}

// UsageProcessingStatusRepository defines the interface for tracking usage processing status
type UsageProcessingStatusRepository interface {
	// Create or update processing status for a billing period
	UpsertProcessingStatus(ctx context.Context, status entities.UsageProcessingStatus) error

	// Get processing status for a billing period
	GetProcessingStatus(ctx context.Context, orgID, subscriptionItemID, billingPeriod string) (entities.UsageProcessingStatus, error)

	// Get all unprocessed usage for billing
	GetUnprocessedUsage(ctx context.Context, orgID, billingPeriod string) ([]entities.UsageProcessingStatus, error)

	// Mark usage as processed with invoice ID
	MarkAsProcessed(ctx context.Context, orgID, subscriptionItemID, billingPeriod, invoiceID string) error

	// Get processing status by invoice
	GetByInvoiceID(ctx context.Context, invoiceID string) ([]entities.UsageProcessingStatus, error)
}

// UsageAggregationRepository defines the interface for usage aggregation operations
type UsageAggregationRepository interface {
	// Get monthly usage aggregates for billing
	GetMonthlyUsage(ctx context.Context, orgID string, billingPeriod time.Time) ([]entities.MonthlyUsageAggregate, error)

	// Get real-time usage summary from materialized views
	GetRealtimeUsage(ctx context.Context, orgID, subscriptionItemID string, 
		since time.Time) (entities.UsageSummary, error)

	// Get customer usage summary
	GetCustomerUsage(ctx context.Context, orgID, customerID string, 
		startTime, endTime time.Time) (entities.CustomerUsageSummary, error)

	// Get usage analytics by type
	GetUsageTypeAnalytics(ctx context.Context, orgID string, 
		startTime, endTime time.Time) ([]entities.UsageTypeAnalytics, error)

	// Refresh materialized views manually (for billing consistency)
	RefreshAggregates(ctx context.Context) error
}

// UsageEventLogRepository defines the interface for usage event logging
type UsageEventLogRepository interface {
	// Create a new log entry
	Create(ctx context.Context, log entities.UsageEventLog) error

	// Find log entries by organization
	FindByOrg(ctx context.Context, orgID string, limit, offset int) ([]entities.UsageEventLog, error)

	// Find log entries by event type
	FindByEventType(ctx context.Context, orgID, eventType string, limit, offset int) ([]entities.UsageEventLog, error)

	// Find log entries by subscription
	FindBySubscription(ctx context.Context, orgID, subscriptionID string, limit, offset int) ([]entities.UsageEventLog, error)

	// Find log entries by invoice
	FindByInvoice(ctx context.Context, orgID, invoiceID string) ([]entities.UsageEventLog, error)
}
