# Aggregate-First Usage Billing Design Guidelines

## Core Concept

The aggregate-first approach separates **what happened** from **what it costs**. Raw usage events are collected and stored without any pricing information, then aggregated and priced only when generating invoices. This creates a clear boundary between usage tracking and financial calculations.

## The Two-Stage Process

### Stage 1: Raw Event Collection
Capture pure usage data as events occur, recording only the operational facts:
- What service was used
- How much was consumed
- When it happened
- Who used it

### Stage 2: Invoice-Time Aggregation and Pricing
When generating an invoice:
1. Query all raw events for the billing period
2. Aggregate usage according to billing requirements
3. Apply current pricing rules to aggregated totals
4. Generate final invoice line items

## Usage Pattern Examples

### Simple Consumption Billing
**Use Case**: API calls, storage bytes, bandwidth usage

**Aggregation Pattern**: Sum total usage across the billing period
- Raw events: Individual API calls with request counts
- Aggregation: Total API calls per month
- Pricing: Apply tiered pricing to total volume

**Example**: A customer makes 1.2 million API calls. Aggregate to monthly total, then apply pricing tiers (first 100k free, next 900k at $0.01 each, remainder at $0.005 each).

### Time-Based Usage
**Use Case**: Compute hours, seat licenses, service uptime

**Aggregation Pattern**: Calculate duration or time-based metrics
- Raw events: Service start/stop events with timestamps
- Aggregation: Total active hours per billing period
- Pricing: Apply hourly rates with potential volume discounts

**Example**: A virtual server runs for varying durations throughout the month. Aggregate total runtime hours, then apply hourly pricing with discounts for high usage.

### Peak Usage Billing
**Use Case**: Maximum concurrent users, peak bandwidth, highest storage

**Aggregation Pattern**: Find maximum values during billing period
- Raw events: Point-in-time usage measurements
- Aggregation: Identify peak usage levels
- Pricing: Bill based on highest consumption point

**Example**: A customer's storage usage fluctuates daily. Find the highest storage amount used during the month and bill based on that peak level.

### Feature-Based Usage
**Use Case**: Premium features, add-on services, per-transaction fees

**Aggregation Pattern**: Count feature usage events and categorize
- Raw events: Feature activation events with context
- Aggregation: Count usage by feature type
- Pricing: Apply different rates per feature category

**Example**: A customer uses basic search (unlimited) and AI-powered search (premium). Count AI search uses separately and apply premium pricing only to those transactions.

### Graduated Pricing Models
**Use Case**: Progressive tiers, volume discounts, usage commitments

**Aggregation Pattern**: Accumulate usage and apply complex pricing logic
- Raw events: Individual usage instances
- Aggregation: Running totals with tier tracking
- Pricing: Apply graduated rates as usage crosses thresholds

**Example**: Email sending service with tiers: first 10k emails at $0.10 each, next 40k at $0.08 each, above 50k at $0.05 each. Aggregate total emails sent, then calculate cost across all applicable tiers.

### Mixed Billing Models
**Use Case**: Base subscription plus usage overages

**Aggregation Pattern**: Combine different aggregation methods
- Raw events: All usage activities
- Aggregation: Multiple aggregation types (sums, peaks, counts)
- Pricing: Apply base rates plus overage calculations

**Example**: A customer has a plan with 1000 included API calls and 10GB included storage. Aggregate API calls and storage separately, calculate overages beyond included amounts, and add to base subscription fee.

## Design Benefits

### Pricing Flexibility
- Change pricing models without touching historical usage data
- Implement complex pricing retroactively
- Test different pricing strategies on the same usage data

### Operational Clarity
- Clear audit trail from raw events to final charges
- Ability to explain any charge by tracing back to source events
- Simplified dispute resolution with transparent calculations

### System Scalability
- Raw events can be optimized for high-volume ingestion
- Pricing calculations only run during invoice generation
- Can implement different storage strategies for hot and cold data

### Business Adaptability
- Support multiple billing cycles for the same usage data
- Handle plan changes mid-cycle without data migration
- Enable pro-rating and partial period billing

## Key Considerations

### Data Retention Strategy
Plan for storing raw events longer than processed invoices to support auditing, disputes, and pricing model changes.

### Performance Management
Design aggregation queries to handle your expected usage volumes efficiently, considering indexing strategies and query optimization.

### Pricing Rule Management
Maintain clear versioning and effective dating of pricing rules to ensure consistent invoice generation over time.

### Edge Case Handling
Consider scenarios like partial periods, plan changes, refunds, and adjustments in your aggregation logic.

## Implementation Philosophy

The aggregate-first approach treats usage data as an immutable historical record and pricing as a flexible business rule applied to that record. This separation enables billing systems that can evolve with changing business requirements while maintaining data integrity and audit capabilities.

The key is to capture usage events with sufficient detail to support any future pricing model, then design aggregation logic that can transform that raw data into the specific metrics needed for your billing approach.


# Raw Events + Async Processing Implementation Specification

## Overview

This specification outlines the implementation of raw event publishing with async processing for usage-based billing using the CloudEvents specification format. The system will publish raw usage events immediately and calculate billing amounts asynchronously through event consumers.


## Architecture Changes

### Event Flow (Aggregate-First Design)
```
API (CloudEvents) → RawUsageEvent Storage → BillingService (Invoice Time)
                                          ↓
                                      Query + Aggregate + Price → Invoice
```

### Key Principles
1. **Raw Event Storage Only**: Events are stored without any pricing calculations
2. **Deferred Aggregation**: All aggregation happens at invoice generation time
3. **Pricing at Bill Time**: Current pricing rules applied when generating invoices
4. **Pure Event Sourcing**: Raw events preserve complete operational history

## CloudEvents Format

We adopt the [CloudEvents specification v1.0](https://cloudevents.io/) for event structure, providing a standardized, extensible format that can grow beyond subscription billing.

### CloudEvents Structure
```json
{
  "specversion": "1.0",
  "type": "api.request",
  "id": "evt_12345",
  "time": "2024-01-01T00:00:00.001Z",
  "source": "billing-api",
  "subject": "sub_item_67890",
  "data": {
    "duration_ms": 150,
    "method": "POST",
    "endpoint": "/api/users",
    "response_size_bytes": 1024
  }
}
```

### Field Mappings
- **subject**: `subscriptionItemId` (the entity being metered)
- **source**: Service/application making the API call (e.g., "billing-api", "user-service")
- **type**: Meter type/id that will process this event
- **id**: Unique event identifier for deduplication
- **data**: Flexible JSON payload with meter-specific data

## Implementation Requirements

### 1. Event Models

#### A. CloudEvents Usage Event
**File**: `internal/application/lib/events/usage_events.go`

```go
package events

import (
    "encoding/json"
    "time"
)

// CloudEventUsageEvent represents raw usage data in CloudEvents format
type CloudEventUsageEvent struct {
    // CloudEvents v1.0 specification fields
    SpecVersion string    `json:"specversion"`           // Always "1.0"
    Type        string    `json:"type"`                  // Meter type/id
    Id          string    `json:"id"`                    // Unique event identifier
    Time        time.Time `json:"time"`                  // Event timestamp (RFC3339)
    Source      string    `json:"source"`                // Service/app that generated the event
    Subject     string    `json:"subject"`               // subscriptionItemId (entity being metered)
    Data        map[string]interface{} `json:"data"`     // Flexible event payload
    
    // Optional CloudEvents fields
    DataContentType string `json:"datacontenttype,omitempty"` // Default: "application/json"
    SchemaURL       string `json:"schemaurl,omitempty"`       // Optional schema reference
}

// RawUsageRecordedEvent represents a raw usage event for storage
type RawUsageRecordedEvent struct {
    BaseEvent
    
    // Original CloudEvent data
    CloudEvent CloudEventUsageEvent `json:"cloud_event"`
    
    // Enriched context (resolved from subject)
    OrgId              string `json:"org_id"`
    SubscriptionId     string `json:"subscription_id"`
    SubscriptionItemId string `json:"subscription_item_id"`
    CustomerId         string `json:"customer_id"`
    MeterId            string `json:"meter_id"`
    
    // Processing metadata
    ReceivedAt time.Time `json:"received_at"`
}
```

### 2. Simplified Event Publisher Interface

**File**: `internal/application/lib/events/publisher.go`

```go
package events

import "context"

type DurableEventPublisher interface {
    // Raw usage events for storage only
    PublishRawUsageEvent(ctx context.Context, event RawUsageRecordedEvent) error
    
    // Generic event publishing
    PublishEvent(ctx context.Context, topic string, event interface{}) error
}
```

### 3. Usage Recording Service Changes

**File**: `internal/application/services/usage_recording_service.go`

```go
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
    subscriptionRepo     repositories.SubscriptionRepository
    subscriptionItemRepo repositories.SubscriptionItemRepository
    meterRepo            repositories.MeterRepository
    durablePublisher     events.DurableEventPublisher
    logger               logger.Logger
}

func NewUsageRecordingService(
    subscriptionRepo repositories.SubscriptionRepository,
    subscriptionItemRepo repositories.SubscriptionItemRepository,
    meterRepo repositories.MeterRepository,
    durablePublisher events.DurableEventPublisher,
    logger logger.Logger,
) interfaces.UsageRecordingService {
    return &UsageRecordingService{
        subscriptionRepo:     subscriptionRepo,
        subscriptionItemRepo: subscriptionItemRepo,
        meterRepo:            meterRepo,
        durablePublisher:     durablePublisher,
        logger:               logger,
    }
}

func (s *UsageRecordingService) RecordCloudEventUsage(
    ctx context.Context,
    orgId string,
    input dto.CloudEventUsageInput,
) (dto.UsageRecordingResponse, error) {
    // 1. Validate CloudEvent format
    if input.CloudEvent.SpecVersion != "1.0" {
        return dto.UsageRecordingResponse{}, fmt.Errorf("unsupported CloudEvents spec version: %s", input.CloudEvent.SpecVersion)
    }
    
    if input.CloudEvent.Subject == "" {
        return dto.UsageRecordingResponse{}, fmt.Errorf("CloudEvent subject (subscriptionItemId) is required")
    }

    // 2. Resolve subject to subscription item
    subscriptionItem, err := s.subscriptionItemRepo.FindById(ctx, orgId, input.CloudEvent.Subject)
    if err != nil {
        return dto.UsageRecordingResponse{}, fmt.Errorf("subscription item not found: %w", err)
    }

    // 3. Validate subscription belongs to org
    subscription, err := s.subscriptionRepo.FindById(ctx, orgId, subscriptionItem.SubscriptionId)
    if err != nil {
        return dto.UsageRecordingResponse{}, fmt.Errorf("subscription not found: %w", err)
    }

    // 4. Validate subscription item has usage enabled and meter configured
    if !subscriptionItem.HasUsage {
        return dto.UsageRecordingResponse{}, fmt.Errorf("subscription item does not support usage recording")
    }
    
    if subscriptionItem.MeterId == "" {
        return dto.UsageRecordingResponse{}, fmt.Errorf("subscription item does not have a meter configured")
    }

    // 5. Validate meter exists and event type matches meter configuration
    meter, err := s.meterRepo.FindById(ctx, orgId, subscriptionItem.MeterId)
    if err != nil {
        return dto.UsageRecordingResponse{}, fmt.Errorf("meter not found: %w", err)
    }
    
    // Match CloudEvent type to meter (can be meter ID or event name)
    if input.CloudEvent.Type != meter.Id && input.CloudEvent.Type != meter.EventName {
        return dto.UsageRecordingResponse{}, fmt.Errorf("CloudEvent type %s does not match meter %s", input.CloudEvent.Type, meter.Id)
    }

    // 6. Set timestamp if not provided
    timestamp := input.CloudEvent.Time
    if timestamp.IsZero() {
        timestamp = time.Now().UTC()
        input.CloudEvent.Time = timestamp
    }

    // 7. Create enriched raw usage event
    eventId := lib.GenerateId("ue")
    
    baseEvent := events.NewBaseEvent(
        orgId,
        events.RawUsageRecorded,
        eventId,
        "raw_usage",
    )

    rawUsageEvent := events.RawUsageRecordedEvent{
        BaseEvent:          baseEvent,
        CloudEvent:         input.CloudEvent,
        OrgId:              orgId,
        SubscriptionId:     subscription.Id,
        SubscriptionItemId: subscriptionItem.Id,
        CustomerId:         subscription.CustomerId,
        MeterId:            subscriptionItem.MeterId,
        ReceivedAt:         time.Now().UTC(),
    }

    // 8. Publish raw usage event for storage
    err = s.durablePublisher.PublishRawUsageEvent(ctx, rawUsageEvent)
    if err != nil {
        s.logger.Error("Failed to publish raw usage event", "error", err)
        return dto.UsageRecordingResponse{}, fmt.Errorf("failed to record usage: %w", err)
    }

    s.logger.Info("Raw usage event published successfully",
        "subscriptionItemId", subscriptionItem.Id,
        "cloudEventType", input.CloudEvent.Type,
        "cloudEventId", input.CloudEvent.Id,
        "eventId", eventId)

    // 9. Return response immediately (no calculations performed)
    return dto.UsageRecordingResponse{
        EventId:            eventId,
        OriginalEventId:    input.CloudEvent.Id,
        SubscriptionItemId: subscriptionItem.Id,
        Type:               input.CloudEvent.Type,
        Status:             "recorded",
        RecordedAt:         timestamp,
    }, nil
}
```

### 4. CloudEvents DTOs

**File**: `internal/application/dto/usage_recording_dto.go`

```go
package dto

import (
    "time"
    "payloop/internal/application/lib/events"
)

// CloudEventUsageInput represents the input for recording CloudEvents usage
type CloudEventUsageInput struct {
    CloudEvent events.CloudEventUsageEvent `json:"cloudevent"`
}

// UsageRecordingResponse represents the immediate response after recording usage
type UsageRecordingResponse struct {
    EventId            string    `json:"event_id"`            // Internal event ID
    OriginalEventId    string    `json:"original_event_id"`   // Original CloudEvent ID
    SubscriptionItemId string    `json:"subscription_item_id"`
    Type               string    `json:"type"`                // CloudEvent type
    Status             string    `json:"status"`              // "recorded", "processing", "calculated"
    RecordedAt         time.Time `json:"recorded_at"`
}
```

### 5. Raw Event Storage Service

**File**: `internal/application/services/raw_usage_storage_service.go`

```go
package services

import (
    "context"
    "fmt"
    
    "payloop/internal/application/lib/events"
    "payloop/internal/application/lib/logger"
    "payloop/internal/domain/repositories"
)

type RawUsageStorageService struct {
    rawUsageRepo repositories.RawUsageRepository
    logger       logger.Logger
}

func NewRawUsageStorageService(
    rawUsageRepo repositories.RawUsageRepository,
    logger logger.Logger,
) *RawUsageStorageService {
    return &RawUsageStorageService{
        rawUsageRepo: rawUsageRepo,
        logger:       logger,
    }
}

func (s *RawUsageStorageService) StoreRawUsageEvent(
    ctx context.Context,
    rawEvent events.RawUsageRecordedEvent,
) error {
    // Simply store the raw event without any calculations
    err := s.rawUsageRepo.Store(ctx, rawEvent)
    if err != nil {
        return fmt.Errorf("failed to store raw usage event: %w", err)
    }

    s.logger.Info("Raw usage event stored successfully",
        "cloudEventId", rawEvent.CloudEvent.Id,
        "subscriptionItemId", rawEvent.SubscriptionItemId,
        "meterId", rawEvent.MeterId)

    return nil
}

```

### 6. Raw Usage Repository

**File**: `internal/domain/repositories/raw_usage_repository.go`

```go
package repositories

import (
    "context"
    "time"
    "payloop/internal/application/lib/events"
)

type RawUsageRepository interface {
    Store(ctx context.Context, event events.RawUsageRecordedEvent) error
    FindById(ctx context.Context, orgId, id string) (events.RawUsageRecordedEvent, error)
    FindBySubscriptionItemId(ctx context.Context, orgId, subscriptionItemId string, startDate, endDate time.Time) ([]events.RawUsageRecordedEvent, error)
    FindByBillingPeriod(ctx context.Context, orgId, subscriptionId, billingPeriod string) ([]events.RawUsageRecordedEvent, error)
    FindByMeter(ctx context.Context, orgId, meterId string, startDate, endDate time.Time) ([]events.RawUsageRecordedEvent, error)
}
```

### 7. Billing Service (Invoice-Time Aggregation)

**File**: `internal/application/services/billing_service.go`

```go
package services

import (
    "context"
    "fmt"
    "time"
    
    "payloop/internal/domain/entities"
    "payloop/internal/domain/repositories"
    "payloop/internal/application/lib/logger"
)

type BillingService struct {
    rawUsageRepo         repositories.RawUsageRepository
    subscriptionRepo     repositories.SubscriptionRepository
    subscriptionItemRepo repositories.SubscriptionItemRepository
    meterRepo            repositories.MeterRepository
    logger               logger.Logger
}

func NewBillingService(
    rawUsageRepo repositories.RawUsageRepository,
    subscriptionRepo repositories.SubscriptionRepository,
    subscriptionItemRepo repositories.SubscriptionItemRepository,
    meterRepo repositories.MeterRepository,
    logger logger.Logger,
) *BillingService {
    return &BillingService{
        rawUsageRepo:         rawUsageRepo,
        subscriptionRepo:     subscriptionRepo,
        subscriptionItemRepo: subscriptionItemRepo,
        meterRepo:            meterRepo,
        logger:               logger,
    }
}

// GenerateUsageCharges aggregates raw events and calculates charges for invoice generation
func (s *BillingService) GenerateUsageCharges(
    ctx context.Context,
    orgId string,
    subscriptionId string,
    billingPeriodStart time.Time,
    billingPeriodEnd time.Time,
) ([]entities.UsageLineItem, error) {
    
    // 1. Get all subscription items with usage billing
    subscription, err := s.subscriptionRepo.FindById(ctx, orgId, subscriptionId)
    if err != nil {
        return nil, fmt.Errorf("failed to find subscription: %w", err)
    }
    
    subscriptionItems, err := s.subscriptionItemRepo.FindBySubscriptionId(ctx, orgId, subscriptionId)
    if err != nil {
        return nil, fmt.Errorf("failed to find subscription items: %w", err)
    }
    
    var usageLineItems []entities.UsageLineItem
    
    // 2. Process each subscription item with usage billing
    for _, item := range subscriptionItems {
        if !item.HasUsage || item.MeterId == "" {
            continue
        }
        
        // 3. Get meter configuration
        meter, err := s.meterRepo.FindById(ctx, orgId, item.MeterId)
        if err != nil {
            s.logger.Warn("Failed to find meter", "meterId", item.MeterId, "error", err)
            continue
        }
        
        // 4. Query raw events for this subscription item and period
        rawEvents, err := s.rawUsageRepo.FindBySubscriptionItemId(
            ctx, orgId, item.Id, billingPeriodStart, billingPeriodEnd,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to query raw events: %w", err)
        }
        
        if len(rawEvents) == 0 {
            continue // No usage for this item
        }
        
        // 5. Aggregate usage based on meter configuration
        aggregatedValue, err := s.aggregateUsage(rawEvents, meter)
        if err != nil {
            return nil, fmt.Errorf("failed to aggregate usage: %w", err)
        }
        
        // 6. Calculate charges based on current pricing
        totalAmount, err := s.calculateCharges(aggregatedValue, item, rawEvents)
        if err != nil {
            return nil, fmt.Errorf("failed to calculate charges: %w", err)
        }
        
        // 7. Create usage line item
        lineItem := entities.UsageLineItem{
            SubscriptionItemId: item.Id,
            MeterId:           meter.Id,
            MeterName:         meter.Name,
            AggregatedValue:   aggregatedValue,
            TotalAmount:       totalAmount,
            EventCount:        len(rawEvents),
            PeriodStart:       billingPeriodStart,
            PeriodEnd:         billingPeriodEnd,
        }
        
        usageLineItems = append(usageLineItems, lineItem)
    }
    
    return usageLineItems, nil
}

// aggregateUsage applies meter aggregation rules to raw events
func (s *BillingService) aggregateUsage(rawEvents []events.RawUsageRecordedEvent, meter entities.Meter) (float64, error) {
    if len(rawEvents) == 0 {
        return 0, nil
    }
    
    var values []float64
    
    // Extract values from each event based on meter configuration
    for _, event := range rawEvents {
        value, err := extractValueFromEventData(event.CloudEvent.Data, meter.ValueProperty)
        if err != nil {
            s.logger.Warn("Failed to extract value from event", 
                "eventId", event.CloudEvent.Id, 
                "valueProperty", meter.ValueProperty,
                "error", err)
            continue
        }
        values = append(values, value)
    }
    
    // Apply aggregation type
    switch meter.AggregationType {
    case entities.AggregationTypeSum:
        var sum float64
        for _, v := range values {
            sum += v
        }
        return sum, nil
        
    case entities.AggregationTypeCount:
        return float64(len(values)), nil
        
    case entities.AggregationTypeMax:
        if len(values) == 0 {
            return 0, nil
        }
        max := values[0]
        for _, v := range values[1:] {
            if v > max {
                max = v
            }
        }
        return max, nil
        
    case entities.AggregationTypeAverage:
        if len(values) == 0 {
            return 0, nil
        }
        var sum float64
        for _, v := range values {
            sum += v
        }
        return sum / float64(len(values)), nil
        
    case entities.AggregationTypeLastDuringPeriod:
        if len(values) == 0 {
            return 0, nil
        }
        // Events should be ordered by time, return last value
        return values[len(values)-1], nil
        
    default:
        return 0, fmt.Errorf("unsupported aggregation type: %s", meter.AggregationType)
    }
}

// calculateCharges applies pricing rules to aggregated usage
func (s *BillingService) calculateCharges(
    aggregatedValue float64, 
    subscriptionItem entities.SubscriptionItem,
    rawEvents []events.RawUsageRecordedEvent,
) (int64, error) {
    
    switch subscriptionItem.UnitType {
    case entities.UnitTypeTransactions:
        // For transaction-based pricing, sum transaction values and apply percentage
        var totalTransactionValue int64
        for _, event := range rawEvents {
            if transactionValue, err := getTransactionValueFromEventData(event.CloudEvent.Data); err == nil {
                totalTransactionValue += transactionValue
            }
        }
        
        // Apply percentage rate
        percentageFee := int64(float64(totalTransactionValue) * subscriptionItem.PercentageRate / 100)
        
        // Add fixed fee per transaction
        fixedFee := int64(aggregatedValue * float64(subscriptionItem.FixedFee))
        
        return percentageFee + fixedFee, nil
        
    default:
        // Unit-based pricing
        return int64(aggregatedValue * float64(subscriptionItem.UnitPrice)), nil
    }
}

// Helper functions for value extraction
func extractValueFromEventData(eventData map[string]interface{}, valueProperty string) (float64, error) {
    // Implementation for extracting values from CloudEvent data
    // This would include the JSONPath-like functionality
    return 0, fmt.Errorf("not implemented")
}

func getTransactionValueFromEventData(eventData map[string]interface{}) (int64, error) {
    // Implementation for extracting transaction values
    return 0, fmt.Errorf("not implemented")
}
```

### 8. Updated Controller for CloudEvents

**File**: `internal/api/controllers/usage_recording_controller.go`

```go
// RecordCloudEventUsage handles POST /api/usage-events (CloudEvents format)
func (u UsageRecordingController) RecordCloudEventUsage(c *gin.Context) {
    user, _ := c.Get("user")
    authUser := user.(authn.User)
    orgId := authUser.OrgId

    var cloudEvent events.CloudEventUsageEvent
    if err := c.ShouldBindJSON(&cloudEvent); err != nil {
        apiErr := api.NewApiError(lib.BadRequestError, "Invalid CloudEvent format", err.Error())
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    // Validate CloudEvent required fields
    if cloudEvent.SpecVersion == "" {
        cloudEvent.SpecVersion = "1.0"
    }
    if cloudEvent.DataContentType == "" {
        cloudEvent.DataContentType = "application/json"
    }

    // Convert to application DTO
    appInput := dto.CloudEventUsageInput{
        CloudEvent: cloudEvent,
    }

    response, err := u.usageRecordingService.RecordCloudEventUsage(c.Request.Context(), orgId, appInput)
    if err != nil {
        apiErr := api.NewApiErrorFromError(err)
        c.JSON(apiErr.GetHttpErrorCode(), apiErr)
        return
    }

    // Convert application response to API response
    apiResponse := mappers.ToCloudEventUsageResponse(response)
    c.JSON(202, apiResponse) // 202 Accepted - processing asynchronously
}
```

### 9. CloudEvents API Response DTO

**File**: `internal/api/dto/response/usage_recording_response.go`

```go
package response

import "time"

type CloudEventUsageResponse struct {
    EventId            string    `json:"event_id"`            // Internal event ID
    OriginalEventId    string    `json:"original_event_id"`   // Original CloudEvent ID
    SubscriptionItemId string    `json:"subscription_item_id"`
    Type               string    `json:"type"`                // CloudEvent type
    Source             string    `json:"source"`              // CloudEvent source
    Status             string    `json:"status"`              // "recorded", "processing", "calculated"
    RecordedAt         time.Time `json:"recorded_at"`
    Message            string    `json:"message,omitempty"`
}
```

### 10. Updated Mappers

**File**: `internal/api/mappers/usage_recording_mapper.go`

```go
package mappers

import (
    "payloop/internal/api/dto/response"
    "payloop/internal/application/dto"
)

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
```

### 11. Event Consumer

**File**: `internal/infrastructure/events/usage_event_consumer.go`

```go
package events

import (
    "context"
    "encoding/json"
    "fmt"
    
    "payloop/internal/application/lib/events"
    "payloop/internal/application/lib/logger"
    "payloop/internal/application/services"
)

type UsageEventConsumer struct {
    calculationService *services.UsageCalculationService
    logger             logger.Logger
}

func NewUsageEventConsumer(
    calculationService *services.UsageCalculationService,
    logger logger.Logger,
) *UsageEventConsumer {
    return &UsageEventConsumer{
        calculationService: calculationService,
        logger:             logger,
    }
}

func (c *UsageEventConsumer) HandleRawUsageEvent(ctx context.Context, message []byte) error {
    var rawEvent events.RawUsageRecordedEvent
    err := json.Unmarshal(message, &rawEvent)
    if err != nil {
        return fmt.Errorf("failed to unmarshal raw usage event: %w", err)
    }

    err = c.calculationService.ProcessRawUsageEvent(ctx, rawEvent)
    if err != nil {
        c.logger.Error("Failed to process raw usage event",
            "eventId", rawEvent.EventId,
            "error", err)
        return err
    }

    c.logger.Info("Raw usage event processed successfully",
        "eventId", rawEvent.EventId)

    return nil
}
```

### 12. Updated Event Types

**File**: `internal/application/lib/events/event_types.go`

```go
package events

const (
    // Raw usage events
    RawUsageRecorded EventType = "raw_usage_recorded"
    
    // Calculated usage events
    UsageCalculated EventType = "usage_calculated"
    
    // Legacy events (deprecated)
    UsageRecorded EventType = "usage_recorded"
)
```

### 13. Simplified CloudEvents Database Schema

**File**: `schemas/usage/schema.prisma`

```prisma
model RawUsageEvent {
  id                 String   @id
  orgId              String   @map("org_id")
  
  // Business context (enriched from CloudEvent subject)
  subscriptionId     String   @map("subscription_id")
  subscriptionItemId String   @map("subscription_item_id")
  customerId         String   @map("customer_id")
  meterId            String   @map("meter_id")
  
  // CloudEvents v1.0 fields
  specVersion String   @map("spec_version")
  type        String   @map("type")
  eventId     String   @map("event_id")      // CloudEvent id field
  time        DateTime @map("time") @db.Timestamptz
  source      String   @map("source")
  subject     String   @map("subject")
  data        Json     @map("data")
  
  // Optional CloudEvents fields
  dataContentType String? @map("data_content_type")
  schemaUrl       String? @map("schema_url")
  
  // Audit
  receivedAt DateTime @default(now()) @map("received_at") @db.Timestamptz
  
  @@index([orgId, subscriptionItemId])
  @@index([orgId, meterId])
  @@index([orgId, subscriptionId])
  @@index([type])
  @@index([source])
  @@index([time])
  @@index([eventId])
  @@index([subscriptionItemId, time])  // Optimized for billing queries
  @@index([meterId, time])             // Optimized for meter queries
  @@map("raw_usage_events")
```

## CloudEvents Benefits

Using CloudEvents specification provides several advantages:

### 1. **Standardization**
- Industry-standard format for event data
- Well-defined schema with consistent field names
- Better interoperability with other systems
- Support for metadata and extensions

### 2. **Extensibility**
- Generic `subject` field allows billing beyond subscriptionItemId
- Can meter customer usage, organization usage, or any entity
- `source` field enables tracking event origins
- Future extensibility for multi-tenant scenarios

### 3. **Flexibility**
- Arbitrary JSON in `data` field supports any meter configuration
- `type` field maps cleanly to meter identifiers
- Optional fields for schema evolution
- Support for event routing and filtering

### 4. **Event Sourcing**
- Complete preservation of original CloudEvent data
- Rich metadata for debugging and analytics
- Deduplication support via CloudEvent `id`
- Traceability across system boundaries

## Implementation Summary

This aggregate-first, CloudEvents-based implementation provides:

### Core Design Benefits
1. **Pure Event Sourcing**: Raw events stored without any calculations
2. **Deferred Aggregation**: All calculations happen at invoice generation time
3. **Pricing Flexibility**: Can change pricing models without touching historical data
4. **Audit Clarity**: Complete trail from raw events to final charges

### Technical Benefits
5. **Industry Standard Format**: CloudEvents v1.0 compliance
6. **Immediate API Response**: Usage recording returns immediately with event ID
7. **Generic Subject Model**: Can extend beyond subscriptionItemId to any billable entity
8. **Scalability**: High-volume usage recording optimized for ingestion
9. **Reliability**: No complex calculations in the critical path
10. **Future-Proof**: Extensible format that can grow with business needs

### Business Benefits
11. **Pricing Model Changes**: Test different pricing on same usage data
12. **Dispute Resolution**: Transparent calculations traceable to source events
13. **Multiple Billing Cycles**: Support different billing periods for same data
14. **Plan Changes**: Handle mid-cycle changes without data migration

The system treats usage data as an immutable historical record and pricing as flexible business rules applied at invoice time. This separation enables billing systems that evolve with changing business requirements while maintaining data integrity and audit capabilities.