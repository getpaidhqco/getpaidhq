package interfaces

import (
	"context"
	"time"

	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

type UsageRecordingService interface {
	// Record single usage event
	RecordUsage(ctx context.Context, input dto.RecordUsageInput) (dto.UsageRecordingResponse, error)

	// Get usage events with pagination
	ListUsageRecords(ctx context.Context, orgId string, input dto.ListUsageRecordsInput) (dto.PaginatedResult[entities.UsageEvent], error)

	// Get specific usage event
	GetUsageEvent(ctx context.Context, orgId string, eventId string) (entities.UsageEvent, error)

	// Get subscription usage by billing period
	GetSubscriptionUsage(ctx context.Context, orgId string, input dto.GetSubscriptionUsageInput) ([]entities.UsageEvent, error)

	// Delete usage event (for corrections)
	DeleteUsageEvent(ctx context.Context, orgId string, eventId string, eventTime time.Time) error

	// Get usage estimate for a subscription
	GetUsageEstimate(ctx context.Context, orgId string, input dto.GetUsageEstimateInput) (dto.UsageEstimateResult, error)
}
