package services

import (
    "context"
    "fmt"
    "time"

    "payloop/internal/api/dto/request"
    "payloop/internal/application/dto"
    "payloop/internal/application/interfaces"
    "payloop/internal/application/lib/logger"
    "payloop/internal/domain/entities"
    "payloop/internal/domain/repositories"
)

type UsageRecordingService struct {
    usageRecordRepo      repositories.UsageRecordRepository
    subscriptionRepo     repositories.SubscriptionRepository
    subscriptionItemRepo repositories.SubscriptionItemRepository
    logger               logger.Logger
}

func NewUsageRecordingService(
    usageRecordRepo repositories.UsageRecordRepository,
    subscriptionRepo repositories.SubscriptionRepository, 
    subscriptionItemRepo repositories.SubscriptionItemRepository,
    logger logger.Logger,
) interfaces.UsageRecordingService {
    return &UsageRecordingService{
        usageRecordRepo:      usageRecordRepo,
        subscriptionRepo:     subscriptionRepo,
        subscriptionItemRepo: subscriptionItemRepo,
        logger:               logger,
    }
}

func (s *UsageRecordingService) RecordUsage(
    ctx context.Context, 
    orgId string, 
    input dto.RecordUsageInput,
) (entities.UsageRecord, error) {
    // 1. Validate subscription item exists and belongs to org
    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, input.SubscriptionItemId)
    if err != nil {
        return entities.UsageRecord{}, fmt.Errorf("subscription item not found: %w", err)
    }

    // 2. Validate subscription belongs to org
    subscription, err := s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return entities.UsageRecord{}, fmt.Errorf("subscription not found: %w", err)
    }

    // 3. Validate subscription item has usage enabled
    if !subscriptionItem.HasUsage {
        return entities.UsageRecord{}, fmt.Errorf("subscription item does not support usage recording")
    }

    // 4. Check for duplicate usage record (idempotency)
    if input.ReferenceId != "" {
        // Since FindByReferenceId is not in the interface, we'll use Find with pagination
        // In a real implementation, you would add a filter for reference_id
        pagination := dto.NewPagination(0, 100, "created_at", "desc")

        // Convert application DTO pagination to request pagination for repository
        repoPagination := request.Pagination{
            Page:          pagination.Page,
            Limit:         pagination.Limit,
            Offset:        pagination.Offset,
            SortDirection: pagination.SortDirection,
            SortBy:        pagination.SortBy,
        }

        existingRecords, _, err := s.usageRecordRepo.Find(ctx, orgId, repoPagination)
        if err == nil {
            // Manually filter for the reference ID
            for _, record := range existingRecords {
                if record.ReferenceId == input.ReferenceId {
                    return record, nil
                }
            }
        }
    }

    // 5. Create appropriate usage record based on subscription item type
    var usageRecord entities.UsageRecord

    switch subscriptionItem.UnitType {
    case entities.UnitTypeTransactions:
        // Transaction-based usage (with value and percentage)
        var transactionValue int64
        var percentageRate float64

        if input.TransactionValue != nil {
            transactionValue = *input.TransactionValue
        }

        if input.PercentageRate != nil {
            percentageRate = *input.PercentageRate
        }

        usageRecord = entities.NewTransactionUsageRecord(
            orgId,
            subscription.Id,
            input.SubscriptionItemId,
            subscription.CustomerId,
            subscriptionItem.PriceId,
            input.Quantity,
            transactionValue,
            percentageRate,
            subscriptionItem.FixedFee,
        )
    default:
        // Unit-based usage (quantity only)
        usageRecord = entities.NewUnitUsageRecord(
            orgId,
            subscription.Id,
            input.SubscriptionItemId,
            subscription.CustomerId,
            subscriptionItem.PriceId,
            input.Quantity,
            subscriptionItem.UnitPrice,
        )
    }

    // 6. Set optional fields
    if input.ReferenceId != "" {
        usageRecord.ReferenceId = input.ReferenceId
        usageRecord.ReferenceType = input.ReferenceType
    }

    if input.Metadata != nil {
        usageRecord.SetMetadata(input.Metadata)
    }

    // Set timestamp if provided
    if !input.Timestamp.IsZero() {
        usageRecord.UsageDate = input.Timestamp
        usageRecord.BillingPeriod = formatBillingPeriod(input.Timestamp)
    }

    // 7. Save usage record
    createdRecord, err := s.usageRecordRepo.Create(ctx, usageRecord)
    if err != nil {
        s.logger.Error("Failed to create usage record", "error", err)
        return entities.UsageRecord{}, fmt.Errorf("failed to record usage: %w", err)
    }

    s.logger.Info("Usage recorded successfully",
        "subscriptionItemId", input.SubscriptionItemId,
        "quantity", input.Quantity,
        "usageRecordId", createdRecord.Id)

    return createdRecord, nil
}

func (s *UsageRecordingService) BatchRecordUsage(
    ctx context.Context,
    orgId string,
    input dto.BatchRecordUsageInput,
) ([]entities.UsageRecord, error) {
    var records []entities.UsageRecord

    // Process each usage record in the batch
    for _, usageInput := range input.Records {
        record, err := s.RecordUsage(ctx, orgId, usageInput)
        if err != nil {
            // Log error but continue processing other records
            s.logger.Error("Failed to record usage in batch",
                "subscriptionItemId", usageInput.SubscriptionItemId,
                "error", err)
            continue
        }
        records = append(records, record)
    }

    return records, nil
}

func (s *UsageRecordingService) ListUsageRecords(
    ctx context.Context,
    orgId string,
    input dto.ListUsageRecordsInput,
) (dto.PaginatedResult[entities.UsageRecord], error) {
    // 1. Validate subscription item belongs to org
    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, input.SubscriptionItemId)
    if err != nil {
        return dto.PaginatedResult[entities.UsageRecord]{}, fmt.Errorf("subscription item not found: %w", err)
    }

    // Verify the subscription exists and belongs to the org
    _, err = s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return dto.PaginatedResult[entities.UsageRecord]{}, fmt.Errorf("subscription not found: %w", err)
    }

    // 2. Get usage records with pagination
    usageRecords, err := s.usageRecordRepo.FindBySubscriptionItemId(ctx, orgId, input.SubscriptionItemId)
    if err != nil {
        return dto.PaginatedResult[entities.UsageRecord]{}, err
    }

    // Apply pagination manually since the repository doesn't support it directly
    pagination := input.Pagination
    offset := pagination.Offset
    limit := pagination.Limit

    total := len(usageRecords)
    start := offset
    end := offset + limit
    if start >= total {
        start = total
    }
    if end > total {
        end = total
    }

    paginatedRecords := usageRecords
    if start < end {
        paginatedRecords = usageRecords[start:end]
    } else {
        paginatedRecords = []entities.UsageRecord{}
    }

    // 3. Create paginated result
    hasMore := (pagination.Page+1)*pagination.Limit < total

    return dto.PaginatedResult[entities.UsageRecord]{
        Items:      paginatedRecords,
        TotalCount: total,
        Page:       pagination.Page,
        PageSize:   pagination.Limit,
        HasMore:    hasMore,
    }, nil
}

func (s *UsageRecordingService) GetUsageRecord(
    ctx context.Context,
    orgId string,
    usageRecordId string,
) (entities.UsageRecord, error) {
    // 1. Get usage record
    usageRecord, err := s.usageRecordRepo.FindById(ctx, orgId, usageRecordId)
    if err != nil {
        return entities.UsageRecord{}, err
    }

    // 2. Validate access through subscription
    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, usageRecord.SubscriptionItemId)
    if err != nil {
        return entities.UsageRecord{}, fmt.Errorf("subscription item not found: %w", err)
    }

    // Verify the subscription exists and belongs to the org
    _, err = s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return entities.UsageRecord{}, fmt.Errorf("subscription not found: %w", err)
    }

    return usageRecord, nil
}

func (s *UsageRecordingService) GetUsageSummary(
    ctx context.Context,
    orgId string,
    input dto.UsageSummaryInput,
) (dto.UsageSummaryResult, error) {
    // 1. Validate subscription item access
    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, input.SubscriptionItemId)
    if err != nil {
        return dto.UsageSummaryResult{}, fmt.Errorf("subscription item not found: %w", err)
    }

    subscription, err := s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return dto.UsageSummaryResult{}, fmt.Errorf("subscription not found: %w", err)
    }

    // 2. Get usage summary from repository
    summaryData, err := s.usageRecordRepo.GetUsageSummary(
        ctx, orgId, input.SubscriptionItemId, input.StartDate, input.EndDate)
    if err != nil {
        return dto.UsageSummaryResult{}, err
    }

    // 3. Create result
    summary := dto.UsageSummaryResult{
        SubscriptionId:     subscription.Id,
        SubscriptionItemId: input.SubscriptionItemId,
        BillingPeriod:      formatBillingPeriod(input.StartDate),
        UsageType:          subscriptionItem.UsageType,
        UnitType:           subscriptionItem.UnitType,
        AggregationType:    subscriptionItem.AggregationType,
        Details:            summaryData,
    }

    // Extract total quantity and amount if available
    if totalQuantity, ok := summaryData["total_quantity"].(float64); ok {
        summary.TotalQuantity = totalQuantity
    }

    if totalAmount, ok := summaryData["total_amount"].(int64); ok {
        summary.TotalAmount = totalAmount
    }

    return summary, nil
}

func (s *UsageRecordingService) GetSubscriptionUsage(
    ctx context.Context,
    orgId string,
    input dto.GetSubscriptionUsageInput,
) ([]entities.UsageRecord, error) {
    // 1. Validate subscription access
    _, err := s.subscriptionRepo.FindById(ctx, orgId, input.SubscriptionId)
    if err != nil {
        return nil, fmt.Errorf("subscription not found: %w", err)
    }

    // 2. Get usage records for billing period
    startPeriod := formatBillingPeriod(input.StartDate)
    endPeriod := formatBillingPeriod(input.EndDate)

    // Use FindByBillingPeriod for each period in the range
    var allRecords []entities.UsageRecord

    // If start and end are in the same period, just do one query
    if startPeriod == endPeriod {
        records, err := s.usageRecordRepo.FindByBillingPeriod(ctx, orgId, input.SubscriptionId, startPeriod)
        if err != nil {
            return nil, err
        }
        allRecords = records
    } else {
        // Otherwise, query each period in the range
        // This is a simplified approach - in a real implementation, you might want to
        // generate all periods between start and end
        recordsStart, err := s.usageRecordRepo.FindByBillingPeriod(ctx, orgId, input.SubscriptionId, startPeriod)
        if err != nil {
            return nil, err
        }

        recordsEnd, err := s.usageRecordRepo.FindByBillingPeriod(ctx, orgId, input.SubscriptionId, endPeriod)
        if err != nil {
            return nil, err
        }

        allRecords = append(recordsStart, recordsEnd...)
    }

    return allRecords, nil
}

func (s *UsageRecordingService) DeleteUsageRecord(
    ctx context.Context,
    orgId string,
    usageRecordId string,
) error {
    // 1. Validate access (same as GetUsageRecord)
    usageRecord, err := s.usageRecordRepo.FindById(ctx, orgId, usageRecordId)
    if err != nil {
        return err
    }

    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, usageRecord.SubscriptionItemId)
    if err != nil {
        return fmt.Errorf("subscription item not found: %w", err)
    }

    // Verify the subscription exists and belongs to the org
    _, err = s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return fmt.Errorf("subscription not found: %w", err)
    }

    // 2. Check if usage record is already processed
    if usageRecord.Processed {
        return fmt.Errorf("cannot delete processed usage record")
    }

    // 3. Delete usage record
    err = s.usageRecordRepo.Delete(ctx, orgId, usageRecordId)
    if err != nil {
        return fmt.Errorf("failed to delete usage record: %w", err)
    }

    s.logger.Info("Usage record deleted", "usageRecordId", usageRecordId)
    return nil
}


// formatBillingPeriod formats the billing period as YYYY-MM
func formatBillingPeriod(date time.Time) string {
    return date.Format("2006-01")
}
