package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
	"time"
)

// UsageRecordRepository defines the interface for usage record repository operations
type UsageRecordRepository interface {
	// FindById finds a usage record by ID
	FindById(ctx context.Context, orgId string, id string) (entities.UsageRecord, error)

	// Create creates a new usage record
	Create(ctx context.Context, entity entities.UsageRecord) (entities.UsageRecord, error)

	// Update updates an existing usage record
	Update(ctx context.Context, entity entities.UsageRecord) (entities.UsageRecord, error)

	// FindBySubscriptionItemId finds all usage records for a subscription item
	FindBySubscriptionItemId(ctx context.Context, orgId string, subscriptionItemId string) ([]entities.UsageRecord, error)

	// FindBySubscriptionId finds all usage records for a subscription
	FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]entities.UsageRecord, error)

	// FindByBillingPeriod finds all usage records for a billing period
	FindByBillingPeriod(ctx context.Context, orgId string, subscriptionId string, billingPeriod string) ([]entities.UsageRecord, error)

	// FindUnprocessed finds all unprocessed usage records
	FindUnprocessed(ctx context.Context, orgId string, subscriptionId string, billingPeriod string) ([]entities.UsageRecord, error)

	// MarkProcessed marks usage records as processed
	MarkProcessed(ctx context.Context, orgId string, ids []string, invoiceId string) error

	// AggregateUsage aggregates usage for a subscription item in a billing period
	AggregateUsage(ctx context.Context, orgId string, subscriptionItemId string, billingPeriod string, aggregationType string) (float64, error)

	// Find finds usage records with pagination
	Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.UsageRecord, int, error)

	// Delete deletes a usage record
	Delete(ctx context.Context, orgId string, id string) error

	// BatchCreate creates multiple usage records in a single transaction
	BatchCreate(ctx context.Context, entities []entities.UsageRecord) ([]entities.UsageRecord, error)

 // GetUsageSummary gets a summary of usage for a subscription item in a billing period
	GetUsageSummary(ctx context.Context, orgId string, subscriptionItemId string, startDate time.Time, endDate time.Time) (map[string]interface{}, error)

	// FindBySubscriptionItem finds all usage records for a subscription item within a date range
	FindBySubscriptionItem(ctx context.Context, orgId string, subscriptionItemId string, startDate time.Time, endDate time.Time) ([]entities.UsageRecord, error)
}
