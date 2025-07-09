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
	timestamp := req.Time
	id := req.Id
	eventType := req.Type
	source := req.Source

	// Set default specversion if not provided
	specVersion := req.SpecVersion
	if specVersion == "" {
		specVersion = "1.0"
	}

	// Create CloudEvents format input
	return dto.RecordUsageInput{
		// OrgId is set by the controller
		SpecVersion: specVersion,
		Type:        eventType,
		Id:          id,
		Time:        timestamp,
		Source:      source,
		Subject:     req.Subject, // subscriptionItemId
		Data:        req.Data,    // Flexible event payload
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
		StartDate:          req.StartDate,
		EndDate:            req.EndDate,
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
		Pagination:         ToPagination(pagination),
	}
}

// ToUsageEventResponse converts domain entity to API response
func ToUsageEventResponse(record entities.UsageEvent) response.UsageEventResponse {
	// Extract values from the Data map
	var quantity float64
	if q, ok := record.Data["quantity"].(float64); ok {
		quantity = q
	}

	var transactionValue int64
	if tv, ok := record.Data["transaction_value"].(float64); ok {
		transactionValue = int64(tv)
	}

	var percentageRate float64
	if pr, ok := record.Data["percentage_rate"].(float64); ok {
		percentageRate = pr
	}

	var referenceId string
	if rid, ok := record.Data["reference_id"].(string); ok {
		referenceId = rid
	}

	var referenceType string
	if rt, ok := record.Data["reference_type"].(string); ok {
		referenceType = rt
	}

	// Convert metadata if present
	metadata := make(map[string]string)
	if record.Metadata != nil {
		metadata = record.Metadata
	}

	// Create response with available fields
	return response.UsageEventResponse{
		OrgId:              record.OrgId,
		Id:                 record.Id,
		SubscriptionId:     record.SubscriptionId,
		SubscriptionItemId: record.SubscriptionItemId,
		MeterId:            record.MeterId,
		UsageType:          entities.UsageTypeMetered, // Default to metered
		Quantity:           quantity,
		TransactionValue:   transactionValue,
		PercentageRate:     percentageRate,
		TotalAmount:        0, // Default to 0
		UsageDate:          record.Time,
		Processed:          false, // Default to false
		ReferenceId:        referenceId,
		ReferenceType:      referenceType,
		Metadata:           metadata,
		CreatedAt:          record.ReceivedAt,
		UpdatedAt:          record.ReceivedAt,
	}
}

// ToUsageEventListResponse converts paginated result to API response
func ToUsageEventListResponse(result dto.PaginatedResult[entities.UsageEvent]) response.UsageEventListResponse {
	items := make([]response.UsageEventResponse, len(result.Items))
	for i, record := range result.Items {
		items[i] = ToUsageEventResponse(record)
	}

	return response.UsageEventListResponse{
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

// ToCloudEventUsageResponse converts application response to API response
func ToCloudEventUsageResponse(appResponse dto.UsageRecordingResponse) response.CloudEventUsageResponse {
	return response.CloudEventUsageResponse{
		EventId:            appResponse.EventId,
		OriginalEventId:    appResponse.OriginalEventId,
		SubscriptionItemId: appResponse.SubscriptionItemId,
		Type:               appResponse.Type,
		Status:             appResponse.Status,
		RecordedAt:         appResponse.RecordedAt,
		Message:            "CloudEvent usage recorded successfully. Calculation in progress.",
	}
}

// Helper function to parse date strings
func parseDate(dateStr string) time.Time {
	// Implementation would depend on the date format used
	// This is a placeholder
	t, _ := time.Parse("2006-01-02", dateStr)
	return t
}
