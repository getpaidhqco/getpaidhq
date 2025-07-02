package interfaces

import (
    "context"

    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
)

type UsageRecordingService interface {
    // Record single usage event
    RecordUsage(ctx context.Context, orgId string, input dto.RecordUsageInput) (entities.UsageRecord, error)

    // Record multiple usage events in batch
    BatchRecordUsage(ctx context.Context, orgId string, input dto.BatchRecordUsageInput) ([]entities.UsageRecord, error)

    // Get usage records with pagination
    ListUsageRecords(ctx context.Context, orgId string, input dto.ListUsageRecordsInput) (dto.PaginatedResult[entities.UsageRecord], error)

    // Get specific usage record
    GetUsageRecord(ctx context.Context, orgId string, usageRecordId string) (entities.UsageRecord, error)

    // Get usage summary for subscription item
    GetUsageSummary(ctx context.Context, orgId string, input dto.UsageSummaryInput) (dto.UsageSummaryResult, error)

    // Get subscription usage by billing period
    GetSubscriptionUsage(ctx context.Context, orgId string, input dto.GetSubscriptionUsageInput) ([]entities.UsageRecord, error)

    // Delete usage record (for corrections)
    DeleteUsageRecord(ctx context.Context, orgId string, usageRecordId string) error
}
