package services

import (
	"context"
	"fmt"
	"time"

	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type UsageRecordingService struct {
	usageEventRepo       repositories.UsageEventRepository
	subscriptionRepo     repositories.SubscriptionRepository
	subscriptionItemRepo repositories.SubscriptionItemRepository
	meterRepo            repositories.MeterRepository
	durablePublisher     events.DurableEventPublisher
	logger               logger.Logger
}

func NewUsageRecordingService(
	usageEventRepo repositories.UsageEventRepository,
	subscriptionRepo repositories.SubscriptionRepository,
	subscriptionItemRepo repositories.SubscriptionItemRepository,
	meterRepo repositories.MeterRepository,
	durablePublisher events.DurableEventPublisher,
	logger logger.Logger,
) interfaces.UsageRecordingService {
	return &UsageRecordingService{
		usageEventRepo:       usageEventRepo,
		subscriptionRepo:     subscriptionRepo,
		subscriptionItemRepo: subscriptionItemRepo,
		meterRepo:            meterRepo,
		durablePublisher:     durablePublisher,
		logger:               logger,
	}
}

// RecordUsage records usage events in CloudEvents format
func (s *UsageRecordingService) RecordUsage(
	ctx context.Context,
	input dto.RecordUsageInput,
) (dto.UsageRecordingResponse, error) {

	if input.Subject == "" {
		return dto.UsageRecordingResponse{}, fmt.Errorf("The subject is required")
	}

	// 2. Resolve subject to subscription item
	subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, input.OrgId, input.Subject)
	if err != nil {
		return dto.UsageRecordingResponse{}, lib.NewCustomError(lib.NotFoundError, "Item not found", err)
	}

	// Validate subscription item has usage enabled and meter configured
	if !subscriptionItem.HasUsage {
		return dto.UsageRecordingResponse{}, fmt.Errorf("subscription item does not support usage recording")
	}

	if subscriptionItem.MeterId == "" {
		return dto.UsageRecordingResponse{}, fmt.Errorf("subscription item does not have a meter configured")
	}

	// 5. Validate meter exists and event type matches meter configuration
	meter, err := s.meterRepo.FindById(ctx, input.OrgId, subscriptionItem.MeterId)
	if err != nil {
		return dto.UsageRecordingResponse{}, fmt.Errorf("meter not found: %w", err)
	}

	// Match CloudEvent type to meter (can be meter ID or event name)
	if input.Type != meter.Id && input.Type != meter.EventName {
		return dto.UsageRecordingResponse{}, fmt.Errorf("CloudEvent type %s does not match meter %s", input.Type, meter.Id)
	}

	// 6. Set timestamp if not provided
	timestamp := input.Time
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
		input.Time = timestamp
	}

	// 7. Create enriched raw usage event
	eventId := lib.GenerateId("ue")

	baseEvent := events.NewBaseEvent(
		input.OrgId,
		events.RawUsageRecorded,
		eventId,
		"raw_usage",
	)

	rawUsageEvent := events.RawUsageRecordedEvent{
		BaseEvent:          baseEvent,
		Data:               input.Data,
		OrgId:              input.OrgId,
		SubscriptionId:     subscriptionItem.SubscriptionId,
		SubscriptionItemId: subscriptionItem.Id,
		MeterId:            subscriptionItem.MeterId,
		ReceivedAt:         time.Now().UTC(),
	}

	// 8. Publish raw usage event for storage
	err = s.durablePublisher.PublishUsageEvent(ctx, rawUsageEvent)
	if err != nil {
		s.logger.Error("Failed to publish raw usage event", "error", err)
		return dto.UsageRecordingResponse{}, fmt.Errorf("failed to record usage: %w", err)
	}

	s.logger.Info("Raw usage event published successfully",
		"subscriptionItemId", subscriptionItem.Id,
		"cloudEventType", input.Type,
		"cloudEventId", input.Id,
		"eventId", eventId)

	// 9. Return response immediately (no calculations performed)
	return dto.UsageRecordingResponse{
		EventId:            eventId,
		OriginalEventId:    input.Id,
		SubscriptionItemId: subscriptionItem.Id,
		Type:               input.Type,
		Status:             "recorded",
		RecordedAt:         timestamp,
	}, nil
}

func (s *UsageRecordingService) ListUsageRecords(
	ctx context.Context,
	orgId string,
	input dto.ListUsageRecordsInput,
) (dto.PaginatedResult[entities.UsageEvent], error) {
	// 1. Validate subscription item belongs to org
	subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, input.SubscriptionItemId)
	if err != nil {
		return dto.PaginatedResult[entities.UsageEvent]{}, fmt.Errorf("subscription item not found: %w", err)
	}

	// Verify the subscription exists and belongs to the org
	_, err = s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
	if err != nil {
		return dto.PaginatedResult[entities.UsageEvent]{}, fmt.Errorf("subscription not found: %w", err)
	}

	// 2. Get usage events with pagination
	// Define time range - for now, get all events (could be refined with start/end time parameters)
	startTime := time.Time{}                        // Zero time for "beginning of time"
	endTime := time.Now().UTC().Add(time.Hour * 24) // Tomorrow, to include all events up to now

	usageEvents, err := s.usageEventRepo.FindBySubscriptionItem(ctx, orgId, input.SubscriptionItemId, startTime, endTime)
	if err != nil {
		return dto.PaginatedResult[entities.UsageEvent]{}, err
	}

	// Apply pagination manually since the repository doesn't support it directly
	pagination := input.Pagination
	offset := pagination.Offset
	limit := pagination.Limit

	total := len(usageEvents)
	start := offset
	end := offset + limit
	if start >= total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedEvents := usageEvents
	if start < end {
		paginatedEvents = usageEvents[start:end]
	} else {
		paginatedEvents = []entities.UsageEvent{}
	}

	// 3. Create paginated result
	hasMore := (pagination.Page+1)*pagination.Limit < total

	return dto.PaginatedResult[entities.UsageEvent]{
		Items:      paginatedEvents,
		TotalCount: total,
		Page:       pagination.Page,
		PageSize:   pagination.Limit,
		HasMore:    hasMore,
	}, nil
}

func (s *UsageRecordingService) GetUsageEvent(
	ctx context.Context,
	orgId string,
	eventId string,
) (entities.UsageEvent, error) {
	// For UsageEvent, we need to find by ID and time, but we don't have time
	// We'll need to modify this to work with the available methods

	// First, try to find by reference ID (which might be the event ID)
	usageEvent, err := s.usageEventRepo.FindByReferenceID(ctx, eventId, "")
	if err != nil {
		// If not found by reference ID, we can't proceed without more information
		return entities.UsageEvent{}, fmt.Errorf("usage event not found: %w", err)
	}

	// 2. Validate access through subscription
	subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, usageEvent.SubscriptionItemId)
	if err != nil {
		return entities.UsageEvent{}, fmt.Errorf("subscription item not found: %w", err)
	}

	// Verify the subscription exists and belongs to the org
	_, err = s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
	if err != nil {
		return entities.UsageEvent{}, fmt.Errorf("subscription not found: %w", err)
	}

	return usageEvent, nil
}

func (s *UsageRecordingService) GetSubscriptionUsage(
	ctx context.Context,
	orgId string,
	input dto.GetSubscriptionUsageInput,
) ([]entities.UsageEvent, error) {
	// 1. Validate subscription access
	_, err := s.subscriptionRepo.FindById(ctx, orgId, input.SubscriptionId)
	if err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	// 2. Get usage events for the date range
	// For UsageEvent, we need to query by time range directly

	// Get all subscription items for this subscription
	subscriptionItems, err := s.subscriptionItemRepo.FindBySubscriptionId(ctx, orgId, input.SubscriptionId)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription items: %w", err)
	}

	var allEvents []entities.UsageEvent

	// For each subscription item, get its usage events
	for _, item := range subscriptionItems {
		// Skip items without usage or meter
		if !item.HasUsage || item.MeterId == "" {
			continue
		}

		// Get usage events for this subscription item
		events, err := s.usageEventRepo.FindBySubscriptionItem(
			ctx, orgId, item.Id, input.StartDate, input.EndDate)
		if err != nil {
			s.logger.Warn("Failed to get usage events for subscription item",
				"subscriptionItemId", item.Id, "error", err)
			continue
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (s *UsageRecordingService) DeleteUsageEvent(
	ctx context.Context,
	orgId string,
	eventId string,
	eventTime time.Time,
) error {
	// For UsageEvent, we need to find by ID and time
	// First, try to find by reference ID (which might be the event ID)
	usageEvent, err := s.usageEventRepo.FindByReferenceID(ctx, eventId, "")
	if err != nil {
		// If not found by reference ID, we can't proceed without more information
		return fmt.Errorf("usage event not found: %w", err)
	}

	subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, usageEvent.SubscriptionItemId)
	if err != nil {
		return fmt.Errorf("subscription item not found: %w", err)
	}

	// Verify the subscription exists and belongs to the org
	_, err = s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
	if err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	// 3. Delete usage event
	err = s.usageEventRepo.Delete(ctx, orgId, usageEvent.SubscriptionItemId, eventTime)
	if err != nil {
		return fmt.Errorf("failed to delete usage event: %w", err)
	}

	s.logger.Info("Usage event deleted", "eventId", eventId)
	return nil
}

// formatBillingPeriod formats the billing period as YYYY-MM
func formatBillingPeriod(date time.Time) string {
	return date.Format("2006-01")
}
