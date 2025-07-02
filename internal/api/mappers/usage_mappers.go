package mappers

import (
    "time"

    "payloop/internal/api/dto/request"
    "payloop/internal/api/dto/response"
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
)

// ToRecordUsageInput converts API request to application input
func ToRecordUsageInput(req request.RecordUsageRequest) dto.RecordUsageInput {
    var transactionValue *int64
    var percentageRate *float64

    // Only set pointers if values are non-zero
    if req.TransactionValue != 0 {
        tv := req.TransactionValue
        transactionValue = &tv
    }

    if req.PercentageRate != 0 {
        pr := req.PercentageRate
        percentageRate = &pr
    }

    return dto.RecordUsageInput{
        SubscriptionItemId: req.SubscriptionItemId,
        Quantity:          req.Quantity,
        TransactionValue:  transactionValue,
        PercentageRate:    percentageRate,
        ReferenceId:       req.ReferenceId,
        ReferenceType:     req.ReferenceType,
        Timestamp:         req.Timestamp,
        Metadata:          req.Metadata,
    }
}

// ToBatchRecordUsageInput converts API request to application input
func ToBatchRecordUsageInput(req request.BatchRecordUsageRequest) dto.BatchRecordUsageInput {
    records := make([]dto.RecordUsageInput, len(req.Records))
    for i, record := range req.Records {
        records[i] = ToRecordUsageInput(record)
    }
    return dto.BatchRecordUsageInput{
        Records: records,
    }
}

// ToUsageSummaryInput converts API request to application input
func ToUsageSummaryInput(req request.GetUsageSummaryRequest) dto.UsageSummaryInput {
    return dto.UsageSummaryInput{
        SubscriptionItemId: req.SubscriptionItemId,
        StartDate:         req.StartDate,
        EndDate:           req.EndDate,
    }
}

// ToGetSubscriptionUsageInput converts API request parameters to application input
func ToGetSubscriptionUsageInput(subscriptionId string, startDate, endDate string) dto.GetSubscriptionUsageInput {
    return dto.GetSubscriptionUsageInput{
        SubscriptionId: subscriptionId,
        StartDate:      parseDate(startDate),
        EndDate:        parseDate(endDate),
    }
}

// ToListUsageRecordsInput converts API request parameters to application input
func ToListUsageRecordsInput(subscriptionItemId string, pagination request.Pagination) dto.ListUsageRecordsInput {
    return dto.ListUsageRecordsInput{
        SubscriptionItemId: subscriptionItemId,
        Pagination:        ToPagination(pagination),
    }
}

// ToUsageRecordResponse converts domain entity to API response
func ToUsageRecordResponse(record entities.UsageRecord) response.UsageRecordResponse {
    return response.UsageRecordResponse{
        OrgId:              record.OrgId,
        Id:                 record.Id,
        SubscriptionId:     record.SubscriptionId,
        SubscriptionItemId: record.SubscriptionItemId,
        CustomerId:         record.CustomerId,
        PriceId:            record.PriceId,
        UsageType:          record.UsageType,
        UnitType:           record.UnitType,
        AggregationType:    record.AggregationType,
        Quantity:           record.Quantity,
        UnitPrice:          record.UnitPrice,
        TransactionValue:   record.TransactionValue,
        PercentageRate:     record.PercentageRate,
        CalculatedFee:      record.CalculatedFee,
        FixedFee:           record.FixedFee,
        TotalAmount:        record.TotalAmount,
        UsageDate:          record.UsageDate,
        BillingPeriod:      record.BillingPeriod,
        Processed:          record.Processed,
        ProcessedAt:        record.ProcessedAt,
        InvoiceId:          record.InvoiceId,
        ReferenceId:        record.ReferenceId,
        ReferenceType:      record.ReferenceType,
        Metadata:           record.Metadata,
        CreatedAt:          record.CreatedAt,
        UpdatedAt:          record.UpdatedAt,
    }
}

// ToUsageRecordListResponse converts paginated result to API response
func ToUsageRecordListResponse(result dto.PaginatedResult[entities.UsageRecord]) response.UsageRecordListResponse {
    items := make([]response.UsageRecordResponse, len(result.Items))
    for i, record := range result.Items {
        items[i] = ToUsageRecordResponse(record)
    }

    return response.UsageRecordListResponse{
        Items:      items,
        TotalCount: result.TotalCount,
        Page:       result.Page,
        PageSize:   result.PageSize,
    }
}

// ToUsageSummaryResponse converts application result to API response
func ToUsageSummaryResponse(result dto.UsageSummaryResult) response.UsageSummaryResponse {
    return response.UsageSummaryResponse{
        SubscriptionId:     result.SubscriptionId,
        SubscriptionItemId: result.SubscriptionItemId,
        BillingPeriod:      result.BillingPeriod,
        UsageType:          result.UsageType,
        UnitType:           result.UnitType,
        AggregationType:    result.AggregationType,
        TotalQuantity:      result.TotalQuantity,
        TotalAmount:        result.TotalAmount,
        Details:            result.Details,
    }
}

// Helper function to parse date strings
func parseDate(dateStr string) time.Time {
    // Implementation would depend on the date format used
    // This is a placeholder
    t, _ := time.Parse("2006-01-02", dateStr)
    return t
}
