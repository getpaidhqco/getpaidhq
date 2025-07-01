package interfaces

import (
    "context"
    "time"
    
    "payloop/internal/api/dto/request"
    "payloop/internal/api/dto/response"
)

type UsageRecordingService interface {
    // Record single usage event
    RecordUsage(ctx context.Context, orgId string, req request.RecordUsageRequest) (response.UsageRecordResponse, error)

    // Record multiple usage events in batch
    BatchRecordUsage(ctx context.Context, orgId string, req request.BatchRecordUsageRequest) (response.UsageRecordListResponse, error)

    // Get usage records with pagination
    ListUsageRecords(ctx context.Context, orgId string, subscriptionItemId string, limit, offset int) (response.UsageRecordListResponse, error)

    // Get specific usage record
    GetUsageRecord(ctx context.Context, orgId string, usageRecordId string) (response.UsageRecordResponse, error)

    // Get usage summary for subscription item
    GetUsageSummary(ctx context.Context, orgId string, req request.GetUsageSummaryRequest) (response.UsageSummaryResponse, error)

    // Get subscription usage by billing period
    GetSubscriptionUsage(ctx context.Context, orgId string, subscriptionId string, startDate, endDate time.Time) (response.UsageRecordListResponse, error)

    // Delete usage record (for corrections)
    DeleteUsageRecord(ctx context.Context, orgId string, usageRecordId string) error
}