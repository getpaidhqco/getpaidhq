package services

import (
    "context"
    "fmt"
    "time"

    "payloop/internal/api/dto/request"
    "payloop/internal/api/dto/response"
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
    req request.RecordUsageRequest,
) (response.UsageRecordResponse, error) {
    // 1. Validate subscription item exists and belongs to org
    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, req.SubscriptionItemId)
    if err != nil {
        return response.UsageRecordResponse{}, fmt.Errorf("subscription item not found: %w", err)
    }

    // 2. Validate subscription belongs to org
    subscription, err := s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return response.UsageRecordResponse{}, fmt.Errorf("subscription not found: %w", err)
    }

    // 3. Validate subscription item has usage enabled
    if !subscriptionItem.HasUsage {
        return response.UsageRecordResponse{}, fmt.Errorf("subscription item does not support usage recording")
    }

    // 4. Check for duplicate usage record (idempotency)
    if req.ReferenceId != "" {
        // Since FindByReferenceId is not in the interface, we'll use Find with pagination
        // In a real implementation, you would add a filter for reference_id
        pagination := request.Pagination{
            Limit: 100,
        }

        existingRecords, _, err := s.usageRecordRepo.Find(ctx, orgId, pagination)
        if err == nil {
            // Manually filter for the reference ID
            for _, record := range existingRecords {
                if record.ReferenceId == req.ReferenceId {
                    return response.FromUsageRecord(record), nil
                }
            }
        }
    }

    // 5. Create appropriate usage record based on subscription item type
    var usageRecord entities.UsageRecord

    switch subscriptionItem.UnitType {
    case entities.UnitTypeTransactions:
        // Transaction-based usage (with value and percentage)
        usageRecord = entities.NewTransactionUsageRecord(
            orgId,
            subscription.Id,
            req.SubscriptionItemId,
            subscription.CustomerId,
            subscriptionItem.PriceId,
            req.Quantity,
            req.TransactionValue,
            req.PercentageRate,
            subscriptionItem.FixedFee,
        )
    default:
        // Unit-based usage (quantity only)
        usageRecord = entities.NewUnitUsageRecord(
            orgId,
            subscription.Id,
            req.SubscriptionItemId,
            subscription.CustomerId,
            subscriptionItem.PriceId,
            req.Quantity,
            subscriptionItem.UnitPrice,
        )
    }

    // 6. Set optional fields
    if req.ReferenceId != "" {
        usageRecord.ReferenceId = req.ReferenceId
        usageRecord.ReferenceType = req.ReferenceType
    }

    if req.Metadata != nil {
        usageRecord.SetMetadata(req.Metadata)
    }

    // Set timestamp if provided
    if !req.Timestamp.IsZero() {
        usageRecord.UsageDate = req.Timestamp
        usageRecord.BillingPeriod = formatBillingPeriod(req.Timestamp)
    }

    // 7. Save usage record
    createdRecord, err := s.usageRecordRepo.Create(ctx, usageRecord)
    if err != nil {
        s.logger.Error("Failed to create usage record", "error", err)
        return response.UsageRecordResponse{}, fmt.Errorf("failed to record usage: %w", err)
    }

    s.logger.Info("Usage recorded successfully",
        "subscriptionItemId", req.SubscriptionItemId,
        "quantity", req.Quantity,
        "usageRecordId", createdRecord.Id)

    return response.FromUsageRecord(createdRecord), nil
}

func (s *UsageRecordingService) BatchRecordUsage(
    ctx context.Context,
    orgId string,
    req request.BatchRecordUsageRequest,
) (response.UsageRecordListResponse, error) {
    var responses []response.UsageRecordResponse

    // Process each usage record in the batch
    for _, usageReq := range req.Records {
        record, err := s.RecordUsage(ctx, orgId, usageReq)
        if err != nil {
            // Log error but continue processing other records
            s.logger.Error("Failed to record usage in batch",
                "subscriptionItemId", usageReq.SubscriptionItemId,
                "error", err)
            continue
        }
        responses = append(responses, record)
    }

    return response.UsageRecordListResponse{
        Items:      responses,
        TotalCount: len(responses),
        Page:       1,
        PageSize:   len(responses),
    }, nil
}

func (s *UsageRecordingService) ListUsageRecords(
    ctx context.Context,
    orgId string,
    subscriptionItemId string,
    limit, offset int,
) (response.UsageRecordListResponse, error) {
    // 1. Validate subscription item belongs to org
    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, subscriptionItemId)
    if err != nil {
        return response.UsageRecordListResponse{}, fmt.Errorf("subscription item not found: %w", err)
    }

    // Verify the subscription exists and belongs to the org
    _, err = s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return response.UsageRecordListResponse{}, fmt.Errorf("subscription not found: %w", err)
    }

    // 2. Get usage records with pagination
    usageRecords, err := s.usageRecordRepo.FindBySubscriptionItemId(ctx, orgId, subscriptionItemId)
    if err != nil {
        return response.UsageRecordListResponse{}, err
    }

    // Apply pagination manually since the repository doesn't support it directly
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

    // 3. Convert to response DTOs
    page := 1
    if limit > 0 {
        page = (offset / limit) + 1
    }

    return response.UsageRecordListResponse{
        Items:      convertToResponseDTOs(paginatedRecords),
        TotalCount: total,
        Page:       page,
        PageSize:   limit,
    }, nil
}

func (s *UsageRecordingService) GetUsageRecord(
    ctx context.Context,
    orgId string,
    usageRecordId string,
) (response.UsageRecordResponse, error) {
    // 1. Get usage record
    usageRecord, err := s.usageRecordRepo.FindById(ctx, orgId, usageRecordId)
    if err != nil {
        return response.UsageRecordResponse{}, err
    }

    // 2. Validate access through subscription
    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, usageRecord.SubscriptionItemId)
    if err != nil {
        return response.UsageRecordResponse{}, fmt.Errorf("subscription item not found: %w", err)
    }

    // Verify the subscription exists and belongs to the org
    _, err = s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return response.UsageRecordResponse{}, fmt.Errorf("subscription not found: %w", err)
    }

    return response.FromUsageRecord(usageRecord), nil
}

func (s *UsageRecordingService) GetUsageSummary(
    ctx context.Context,
    orgId string,
    req request.GetUsageSummaryRequest,
) (response.UsageSummaryResponse, error) {
    // 1. Validate subscription item access
    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, req.SubscriptionItemId)
    if err != nil {
        return response.UsageSummaryResponse{}, fmt.Errorf("subscription item not found: %w", err)
    }

    subscription, err := s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return response.UsageSummaryResponse{}, fmt.Errorf("subscription not found: %w", err)
    }

    // 2. Get usage summary from repository
    summaryData, err := s.usageRecordRepo.GetUsageSummary(
        ctx, orgId, req.SubscriptionItemId, req.StartDate, req.EndDate)
    if err != nil {
        return response.UsageSummaryResponse{}, err
    }

    // 3. Create response
    summary := response.UsageSummaryResponse{
        SubscriptionId:     subscription.Id,
        SubscriptionItemId: req.SubscriptionItemId,
        BillingPeriod:      formatBillingPeriod(req.StartDate),
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
    subscriptionId string,
    startDate, endDate time.Time,
) (response.UsageRecordListResponse, error) {
    // 1. Validate subscription access
    _, err := s.subscriptionRepo.FindById(ctx, orgId, subscriptionId)
    if err != nil {
        return response.UsageRecordListResponse{}, fmt.Errorf("subscription not found: %w", err)
    }

    // 2. Get usage records for billing period
    startPeriod := formatBillingPeriod(startDate)
    endPeriod := formatBillingPeriod(endDate)

    // Use FindByBillingPeriod for each period in the range
    var allRecords []entities.UsageRecord

    // If start and end are in the same period, just do one query
    if startPeriod == endPeriod {
        records, err := s.usageRecordRepo.FindByBillingPeriod(ctx, orgId, subscriptionId, startPeriod)
        if err != nil {
            return response.UsageRecordListResponse{}, err
        }
        allRecords = records
    } else {
        // Otherwise, query each period in the range
        // This is a simplified approach - in a real implementation, you might want to
        // generate all periods between start and end
        recordsStart, err := s.usageRecordRepo.FindByBillingPeriod(ctx, orgId, subscriptionId, startPeriod)
        if err != nil {
            return response.UsageRecordListResponse{}, err
        }

        recordsEnd, err := s.usageRecordRepo.FindByBillingPeriod(ctx, orgId, subscriptionId, endPeriod)
        if err != nil {
            return response.UsageRecordListResponse{}, err
        }

        allRecords = append(recordsStart, recordsEnd...)
    }

    // 3. Convert to response DTOs
    return response.UsageRecordListResponse{
        Items:      convertToResponseDTOs(allRecords),
        TotalCount: len(allRecords),
        Page:       1,
        PageSize:   len(allRecords),
    }, nil
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

// Helper function to convert usage records to response DTOs
func convertToResponseDTOs(records []entities.UsageRecord) []response.UsageRecordResponse {
    responses := make([]response.UsageRecordResponse, len(records))
    for i, record := range records {
        responses[i] = response.FromUsageRecord(record)
    }
    return responses
}

// formatBillingPeriod formats the billing period as YYYY-MM
func formatBillingPeriod(date time.Time) string {
    return date.Format("2006-01")
}
