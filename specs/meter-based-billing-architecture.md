# Meter-Based Billing Architecture Specification

## Overview

This specification outlines the implementation of a meter-based billing architecture that separates usage measurement (meters) from pricing. This approach provides flexibility in pricing strategies while maintaining consistent usage tracking.

## Core Concepts

### Meter
A **Meter** defines HOW to measure usage. It specifies:
- What events to track
- How to filter relevant events
- How to aggregate values
- How to display the usage

**Meters are referenced** because they are measurement definitions that can evolve over time to improve accuracy while maintaining consistency.

### Price
A **Price** defines HOW MUCH to charge for measured usage. It references a meter and specifies:
- The cost per unit
- Percentage rates
- Fixed fees
- Included usage allowances

**Prices are copied/snapshotted** because they represent contractual terms that should be locked in when a customer subscribes to maintain pricing stability and enable grandfathering.

## Copy vs Reference Strategy

### Meter = Reference (Mutable Definition)
- **Rationale**: Measurement logic can improve over time
- **Benefits**: Bug fixes, accuracy improvements, shared logic
- **Examples**: Better aggregation algorithms, improved event filtering

### Price = Copy (Immutable Contract)
- **Rationale**: Pricing terms are contractual and should be locked
- **Benefits**: Grandfathering, price stability, compliance
- **Examples**: Customer keeps "$0.01/call" rate even if price changes to "$0.02/call"

## Architecture Changes

### 1. New Meter Entity

Create a new Meter entity that encapsulates all usage measurement logic.

#### Meter Entity Structure

```go
// internal/domain/entities/meter.go
package entities

import (
    "time"
    "payloop/internal/lib"
)

type Meter struct {
    OrgId           string                 `json:"org_id"`
    Id              string                 `json:"id"`
    Slug            string                 `json:"slug"`           // Unique machine-readable identifier (e.g., "api_calls", "storage_gb_hours")
    Name            string                 `json:"name"`           // Human-readable name (e.g., "API Calls", "Storage Usage")
    Description     string                 `json:"description"`    // Detailed description of what this meter measures
    
    // Event Configuration
    EventName       string                 `json:"event_name"`     // The event type to track (e.g., "api.request", "storage.snapshot")
    EventFilter     map[string]interface{} `json:"event_filter"`   // Optional filters to apply (e.g., {"method": "POST", "tier": "premium"})
    
    // Aggregation Configuration
    AggregationType AggregationType        `json:"aggregation_type"` // How to aggregate: sum, count, max, average, last_during_period
    ValueProperty   string                 `json:"value_property"`   // Which event property to aggregate (e.g., "duration", "bytes", "tokens")
    
    // Display Configuration
    UnitType        UnitType               `json:"unit_type"`        // Unit of measurement: gb_hours, api_calls, minutes, etc.
    DisplayName     string                 `json:"display_name"`     // How to display in invoices/UI (e.g., "API Calls", "GB-Hours")
    
    // Window Configuration
    WindowSize      string                 `json:"window_size"`      // Aggregation window: "minute", "hour", "day", "month"
    ResetInterval   string                 `json:"reset_interval"`   // When to reset counters: "hourly", "daily", "monthly", "never"
    
    Metadata        map[string]string      `json:"metadata"`
    CreatedAt       time.Time              `json:"created_at"`
    UpdatedAt       time.Time              `json:"updated_at"`
}

// Factory function
func NewMeter(orgId string, input CreateMeterInput) (Meter, error) {
    if err := validateMeterInput(input); err != nil {
        return Meter{}, err
    }
    
    return Meter{
        OrgId:           orgId,
        Id:              lib.GenerateId("meter"),
        Slug:            input.Slug,
        Name:            input.Name,
        Description:     input.Description,
        EventName:       input.EventName,
        EventFilter:     input.EventFilter,
        AggregationType: input.AggregationType,
        ValueProperty:   input.ValueProperty,
        UnitType:        input.UnitType,
        DisplayName:     input.DisplayName,
        WindowSize:      input.WindowSize,
        ResetInterval:   input.ResetInterval,
        Metadata:        input.Metadata,
        CreatedAt:       time.Now().UTC(),
        UpdatedAt:       time.Now().UTC(),
    }, nil
}
```

#### Field Explanations

**slug**: A unique, machine-readable identifier within an organization. Used in API calls and internal references. Examples:
- `api_calls` - for tracking API requests
- `storage_gb_hours` - for storage usage
- `ai_tokens_gpt4` - for specific AI model usage

**eventName**: The event type that this meter tracks. These are custom strings defined by your application. Common patterns:
- Dot notation: `api.request`, `storage.snapshot`, `ai.completion`
- Underscore notation: `user_login`, `file_upload`, `video_transcode`
- Should be consistent across your application

**eventFilter**: JSON object to filter which events to include. Examples:
```json
// Only track POST requests
{"method": "POST"}

// Only track premium tier storage
{"tier": "premium", "region": "us-east-1"}

// Only track specific AI model
{"model": "gpt-4", "type": "completion"}
```

**valueProperty**: The field in the event to aggregate. If empty, counts events. Examples:
- `duration` - for time-based metrics
- `bytes` - for data transfer
- `tokens` - for AI usage
- `amount` - for financial transactions
- Empty string - just count the events

**windowSize**: Defines the granularity of aggregation:
- `minute` - Aggregate by minute (real-time dashboards)
- `hour` - Aggregate by hour (detailed analytics)
- `day` - Aggregate by day (daily summaries)
- `month` - Aggregate by month (billing periods)

**resetInterval**: When to reset usage counters:
- `hourly` - Reset every hour (rate limiting)
- `daily` - Reset every day (daily quotas)
- `monthly` - Reset every billing month (monthly allowances)
- `never` - Cumulative usage (lifetime metrics)

**displayName**: Human-friendly name shown in invoices and UI:
- `"API Calls"` instead of `api_calls`
- `"Storage (GB-Hours)"` instead of `storage_gb_hours`
- `"GPT-4 Tokens"` instead of `ai_tokens_gpt4`

### 2. Event Name Constants

Define event names as constants to ensure consistency:

```go
// internal/domain/events/event_names.go
package events

// API Events
const (
    EventAPIRequest      = "api.request"
    EventAPIRateLimit    = "api.rate_limit"
    EventAPIError        = "api.error"
)

// Storage Events
const (
    EventStorageSnapshot = "storage.snapshot"
    EventStorageUpload   = "storage.upload"
    EventStorageDelete   = "storage.delete"
)

// AI/ML Events
const (
    EventAICompletion    = "ai.completion"
    EventAIEmbedding     = "ai.embedding"
    EventAIImageGen      = "ai.image_generation"
)

// Transaction Events
const (
    EventPaymentProcess  = "payment.process"
    EventTransferFunds   = "transfer.funds"
    EventRefundIssue     = "refund.issue"
)

// User Activity Events
const (
    EventUserLogin       = "user.login"
    EventUserAction      = "user.action"
    EventUserExport      = "user.export"
)
```

### 3. Update Price Entity

Modify the Price entity to reference meters instead of embedding usage configuration:

```go
// internal/domain/entities/price.go
type Price struct {
    OrgId              string                 `json:"org_id"`
    Id                 string                 `json:"id"`
    VariantId          string                 `json:"variant_id"`
    Label              string                 `json:"label"`
    Category           prices.PriceCategory   `json:"category"`
    Scheme             prices.PriceScheme     `json:"scheme"`
    Cycles             int                    `json:"cycles"`
    Currency           common.Currency        `json:"currency"`
    UnitPrice          int64                  `json:"unit_price"`
    MinPrice           int64                  `json:"min_price"`
    SuggestedPrice     int64                  `json:"suggested_price"`
    BillingInterval    prices.BillingInterval `json:"billing_interval"`
    BillingIntervalQty int                    `json:"billing_interval_qty"`
    TrialInterval      prices.BillingInterval `json:"trial_interval"`
    TrialIntervalQty   int                    `json:"trial_interval_qty"`
    TaxCode            string                 `json:"tax_code"`

    // Meter reference (NEW)
    MeterId            string                 `json:"meter_id,omitempty"`
    
    // Usage-based pricing configuration (KEEP - these define pricing, not measurement)
    HasUsage           bool                   `json:"has_usage"`
    PercentageRate     float64                `json:"percentage_rate,omitempty"`
    FixedFee           int64                  `json:"fixed_fee,omitempty"`
    OverageUnitPrice   int64                  `json:"overage_unit_price,omitempty"`
    IncludedUsage      int64                  `json:"included_usage,omitempty"`
    UsageLimit         int64                  `json:"usage_limit,omitempty"`
    
    // REMOVE: UsageType, UnitType, AggregationType (these move to Meter)

    Tiers              []PriceTier            `json:"tiers,omitempty"`
    Metadata           map[string]string      `json:"metadata"`
    CreatedAt          time.Time              `json:"created_at"`
    UpdatedAt          time.Time              `json:"updated_at"`
}
```

### 4. Update SubscriptionItem Entity

```go
// internal/domain/entities/subscription_item.go
type SubscriptionItem struct {
    OrgId          string                `json:"org_id"`
    Id             string                `json:"id"`
    SubscriptionId string                `json:"subscription_id"`
    Subscription   *Subscription         `json:"-"`

    // Product/Price reference
    PriceId        string                `json:"price_id"`
    ProductId      string                `json:"product_id,omitempty"`
    VariantId      string                `json:"variant_id,omitempty"`
    
    // Meter reference (NEW - referenced, not copied)
    MeterId        string                `json:"meter_id,omitempty"`

    // Item details
    Name           string                `json:"name"`
    Description    string                `json:"description,omitempty"`
    Status         SubscriptionItemStatus `json:"status"`

    // Quantity for fixed items
    Quantity       int                   `json:"quantity"`

    // Billing
    Amount         int64                 `json:"amount,omitempty"`
    Currency       string                `json:"currency"`

    // Pricing configuration (COPIED from Price at creation time for grandfathering)
    HasUsage           bool                  `json:"has_usage"`
    PercentageRate     float64               `json:"percentage_rate,omitempty"`
    FixedFee           int64                 `json:"fixed_fee,omitempty"`
    UnitPrice          int64                 `json:"unit_price,omitempty"`
    OverageUnitPrice   int64                 `json:"overage_unit_price,omitempty"`
    IncludedUsage      int64                 `json:"included_usage,omitempty"`
    UsageLimit         int64                 `json:"usage_limit,omitempty"`
    
    // Price snapshot for comparison/audit (NEW - optional for evaluation)
    PriceSnapshot      json.RawMessage       `json:"price_snapshot,omitempty"`

    // Metadata
    Metadata       map[string]string     `json:"metadata,omitempty"`
    CreatedAt      time.Time             `json:"created_at"`
    UpdatedAt      time.Time             `json:"updated_at"`
}
```

### 5. Database Schema Updates

#### Add Meter Table

```prisma
// prisma/schema.prisma

model Meter {
  orgId           String   @map("org_id")
  id              String   @id @default(cuid())
  slug            String
  name            String
  description     String?
  
  // Event configuration
  eventName       String   @map("event_name")
  eventFilter     Json?    @map("event_filter")
  
  // Aggregation configuration
  aggregationType String   @map("aggregation_type")
  valueProperty   String?  @map("value_property")
  
  // Display configuration
  unitType        String   @map("unit_type")
  displayName     String   @map("display_name")
  
  // Window configuration
  windowSize      String?  @map("window_size")
  resetInterval   String?  @map("reset_interval")
  
  metadata        Json?
  createdAt       DateTime @default(now()) @map("created_at")
  updatedAt       DateTime @updatedAt @map("updated_at")
  
  // Relations
  Org             Org      @relation(fields: [orgId], references: [id])
  Price           Price[]
  SubscriptionItem SubscriptionItem[]
  UsageRecord     UsageRecord[]
  
  @@unique([orgId, slug])
  @@index([orgId, eventName])
  @@map("meters")
}
```

#### Update Price Table

```prisma
model Price {
  // ... existing fields ...
  
  // Add meter reference
  meterId         String?  @map("meter_id")
  Meter           Meter?   @relation(fields: [meterId], references: [id])
  
  // Remove: usageType, unitType, aggregationType
}
```

#### Update SubscriptionItem Table

```prisma
model SubscriptionItem {
  // ... existing fields ...
  
  // Add meter reference
  meterId         String?  @map("meter_id")
  Meter           Meter?   @relation(fields: [meterId], references: [id])
  
  // Add pricing fields (copied from Price at creation time)
  hasUsage        Boolean  @default(false) @map("has_usage")
  percentageRate  Float?   @map("percentage_rate")
  fixedFee        BigInt?  @map("fixed_fee")
  overageUnitPrice BigInt? @map("overage_unit_price")
  includedUsage   BigInt?  @map("included_usage")
  usageLimit      BigInt?  @map("usage_limit")
  
  // Add price snapshot column (for evaluation)
  priceSnapshot   Json?    @map("price_snapshot")
  
  // Remove: usageType, unitType, aggregationType (get from meter)
}
```

#### Update UsageRecord Table

```prisma
model UsageRecord {
  // ... existing fields ...
  
  // Add meter reference
  meterId         String   @map("meter_id")
  Meter           Meter    @relation(fields: [meterId], references: [id])
  
  // Remove: usageType (get from meter)
}
```

### 6. Service Layer Implementation

#### Meter Service

```go
// internal/application/services/meter_service.go
package services

import (
    "context"
    "fmt"
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
    "payloop/internal/domain/repositories"
)

type MeterService interface {
    Create(ctx context.Context, orgId string, input dto.CreateMeterInput) (entities.Meter, error)
    Update(ctx context.Context, orgId, meterId string, input dto.UpdateMeterInput) (entities.Meter, error)
    Get(ctx context.Context, orgId, meterId string) (entities.Meter, error)
    GetBySlug(ctx context.Context, orgId, slug string) (entities.Meter, error)
    List(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.Meter], error)
    Delete(ctx context.Context, orgId, meterId string) error
    ValidateEventAgainstMeter(ctx context.Context, meter entities.Meter, event map[string]interface{}) bool
}

type meterService struct {
    meterRepo repositories.MeterRepository
    logger    logger.Logger
}

func NewMeterService(meterRepo repositories.MeterRepository, logger logger.Logger) MeterService {
    return &meterService{
        meterRepo: meterRepo,
        logger:    logger,
    }
}

func (s *meterService) Create(ctx context.Context, orgId string, input dto.CreateMeterInput) (entities.Meter, error) {
    // Check if slug already exists
    existing, err := s.meterRepo.FindBySlug(ctx, orgId, input.Slug)
    if err == nil && existing.Id != "" {
        return entities.Meter{}, fmt.Errorf("meter with slug %s already exists", input.Slug)
    }
    
    // Create meter entity
    meter, err := entities.NewMeter(orgId, entities.CreateMeterInput{
        Slug:            input.Slug,
        Name:            input.Name,
        Description:     input.Description,
        EventName:       input.EventName,
        EventFilter:     input.EventFilter,
        AggregationType: input.AggregationType,
        ValueProperty:   input.ValueProperty,
        UnitType:        input.UnitType,
        DisplayName:     input.DisplayName,
        WindowSize:      input.WindowSize,
        ResetInterval:   input.ResetInterval,
        Metadata:        input.Metadata,
    })
    if err != nil {
        return entities.Meter{}, err
    }
    
    // Save to repository
    return s.meterRepo.Create(ctx, meter)
}

func (s *meterService) ValidateEventAgainstMeter(ctx context.Context, meter entities.Meter, event map[string]interface{}) bool {
    // Check if event matches meter's event name
    eventName, ok := event["event_name"].(string)
    if !ok || eventName != meter.EventName {
        return false
    }
    
    // Apply event filters if any
    if meter.EventFilter != nil && len(meter.EventFilter) > 0 {
        for key, expectedValue := range meter.EventFilter {
            actualValue, exists := event[key]
            if !exists || actualValue != expectedValue {
                return false
            }
        }
    }
    
    return true
}
```

#### Updated Usage Recording Service

```go
// internal/application/services/usage_recording_service.go
func (s *UsageRecordingService) RecordUsage(ctx context.Context, orgId string, input dto.RecordUsageInput) error {
    // Get subscription item with meter
    subscriptionItem, err := s.subscriptionRepo.GetSubscriptionItemWithMeter(ctx, orgId, input.SubscriptionItemID)
    if err != nil {
        return fmt.Errorf("subscription item not found: %w", err)
    }
    
    if subscriptionItem.MeterId == "" {
        return fmt.Errorf("subscription item does not have a meter configured")
    }
    
    // Get meter details
    meter, err := s.meterService.Get(ctx, orgId, subscriptionItem.MeterId)
    if err != nil {
        return fmt.Errorf("meter not found: %w", err)
    }
    
    // Create usage event based on meter configuration
    event := events.UsageEvent{
        EventID:            uuid.New().String(),
        EventType:          "usage.recorded",
        EventVersion:       "1.0",
        Timestamp:          time.Now(),
        OrgID:              orgId,
        SubscriptionID:     input.SubscriptionID,
        SubscriptionItemID: input.SubscriptionItemID,
        CustomerID:         input.CustomerID,
        MeterId:            meter.Id,
        EventName:          meter.EventName,
        EventData: map[string]interface{}{
            "event_name":   meter.EventName,
            meter.ValueProperty: input.Quantity,
            "reference_id": input.ReferenceID,
            "reference_type": input.ReferenceType,
        },
        Metadata:           input.Metadata,
    }
    
    // Calculate amount based on subscription item pricing
    var calculatedAmount int64
    if input.TransactionValue > 0 && subscriptionItem.PercentageRate > 0 {
        // Percentage-based pricing
        calculatedAmount = int64(float64(input.TransactionValue) * subscriptionItem.PercentageRate / 100.0)
    } else if subscriptionItem.UnitPrice > 0 {
        // Unit-based pricing
        calculatedAmount = int64(input.Quantity * float64(subscriptionItem.UnitPrice))
    }
    
    if subscriptionItem.FixedFee > 0 {
        calculatedAmount += subscriptionItem.FixedFee
    }
    
    event.CalculatedAmount = calculatedAmount
    
    // Publish to Kafka
    return s.kafkaProducer.PublishUsageEvent(ctx, event)
}
```

### 7. API Layer Updates

#### Meter Controller

```go
// internal/api/controllers/meter_controller.go
package controllers

type MeterController struct {
    meterService application.MeterService
}

// Create a new meter
// POST /api/meters
func (c *MeterController) Create(ctx *gin.Context) {
    var req request.CreateMeterRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    orgId := auth.GetOrgId(ctx)
    
    input := dto.CreateMeterInput{
        Slug:            req.Slug,
        Name:            req.Name,
        Description:     req.Description,
        EventName:       req.EventName,
        EventFilter:     req.EventFilter,
        AggregationType: req.AggregationType,
        ValueProperty:   req.ValueProperty,
        UnitType:        req.UnitType,
        DisplayName:     req.DisplayName,
        WindowSize:      req.WindowSize,
        ResetInterval:   req.ResetInterval,
        Metadata:        req.Metadata,
    }
    
    meter, err := c.meterService.Create(ctx, orgId, input)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    ctx.JSON(201, mappers.ToMeterResponse(meter))
}

// List meters
// GET /api/meters
func (c *MeterController) List(ctx *gin.Context) {
    orgId := auth.GetOrgId(ctx)
    pagination := getPagination(ctx)
    
    result, err := c.meterService.List(ctx, orgId, pagination)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    ctx.JSON(200, result)
}

// Get meter by ID or slug
// GET /api/meters/:id
func (c *MeterController) Get(ctx *gin.Context) {
    orgId := auth.GetOrgId(ctx)
    identifier := ctx.Param("id")
    
    var meter entities.Meter
    var err error
    
    // Check if identifier looks like an ID or slug
    if strings.HasPrefix(identifier, "meter_") {
        meter, err = c.meterService.Get(ctx, orgId, identifier)
    } else {
        meter, err = c.meterService.GetBySlug(ctx, orgId, identifier)
    }
    
    if err != nil {
        ctx.JSON(404, gin.H{"error": "meter not found"})
        return
    }
    
    ctx.JSON(200, mappers.ToMeterResponse(meter))
}
```

#### Updated Price Controller

```go
// internal/api/controllers/price_controller.go

// Create price with meter reference
// POST /api/prices
func (c *PriceController) Create(ctx *gin.Context) {
    var req request.CreatePriceRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    orgId := auth.GetOrgId(ctx)
    
    // Validate meter exists if this is a usage-based price
    if req.HasUsage && req.MeterId == "" {
        ctx.JSON(400, gin.H{"error": "meter_id is required for usage-based pricing"})
        return
    }
    
    if req.MeterId != "" {
        _, err := c.meterService.Get(ctx, orgId, req.MeterId)
        if err != nil {
            ctx.JSON(400, gin.H{"error": "invalid meter_id"})
            return
        }
    }
    
    input := dto.CreatePriceInput{
        VariantId:          req.VariantId,
        Label:              req.Label,
        Category:           req.Category,
        Scheme:             req.Scheme,
        Currency:           req.Currency,
        UnitPrice:          req.UnitPrice,
        BillingInterval:    req.BillingInterval,
        HasUsage:           req.HasUsage,
        MeterId:            req.MeterId,
        PercentageRate:     req.PercentageRate,
        FixedFee:           req.FixedFee,
        OverageUnitPrice:   req.OverageUnitPrice,
        IncludedUsage:      req.IncludedUsage,
        UsageLimit:         req.UsageLimit,
        Metadata:           req.Metadata,
    }
    
    price, err := c.priceService.Create(ctx, orgId, input)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    ctx.JSON(201, mappers.ToPriceResponse(price))
}
```

### 8. Example Meter Configurations

```go
// Example 1: API Calls Meter
{
    "slug": "api_calls",
    "name": "API Calls",
    "description": "Tracks all API requests",
    "event_name": "api.request",
    "aggregation_type": "count",
    "unit_type": "count",
    "display_name": "API Calls",
    "window_size": "hour",
    "reset_interval": "monthly"
}

// Example 2: Storage Meter
{
    "slug": "storage_gb_hours",
    "name": "Storage Usage",
    "description": "Tracks storage usage in GB-hours",
    "event_name": "storage.snapshot",
    "aggregation_type": "average",
    "value_property": "bytes_used",
    "unit_type": "gb_hours",
    "display_name": "Storage (GB-Hours)",
    "window_size": "hour",
    "reset_interval": "never"
}

// Example 3: AI Token Meter with Filtering
{
    "slug": "ai_tokens_gpt4",
    "name": "GPT-4 Token Usage",
    "description": "Tracks GPT-4 token consumption",
    "event_name": "ai.completion",
    "event_filter": {
        "model": "gpt-4"
    },
    "aggregation_type": "sum",
    "value_property": "tokens",
    "unit_type": "count",
    "display_name": "GPT-4 Tokens",
    "window_size": "day",
    "reset_interval": "monthly"
}

// Example 4: Transaction Volume Meter
{
    "slug": "payment_volume",
    "name": "Payment Volume",
    "description": "Tracks total payment volume for percentage-based fees",
    "event_name": "payment.process",
    "aggregation_type": "sum",
    "value_property": "amount",
    "unit_type": "cents",
    "display_name": "Payment Volume",
    "window_size": "month",
    "reset_interval": "monthly"
}
```

### 9. Usage Event Structure Update

```go
// internal/domain/events/usage_events.go
type UsageEvent struct {
    // Event metadata
    EventID       string    `json:"event_id"`
    EventType     string    `json:"event_type"`
    EventVersion  string    `json:"event_version"`  // Schema version for event evolution (e.g., "1.0", "1.1", "2.0")
    Timestamp     time.Time `json:"timestamp"`
    
    // Business context
    OrgID              string `json:"org_id"`
    SubscriptionID     string `json:"subscription_id"`
    SubscriptionItemID string `json:"subscription_item_id"`
    CustomerID         string `json:"customer_id"`
    
    // Meter reference (NEW)
    MeterId            string `json:"meter_id"`
    EventName          string `json:"event_name"`
    
    // Event data (NEW - flexible structure based on meter)
    EventData          map[string]interface{} `json:"event_data"`
    
    // Calculated amount (still needed for billing)
    CalculatedAmount   int64  `json:"calculated_amount"`
    
    // Event sourcing metadata
    CausationID        string            `json:"causation_id,omitempty"`
    CorrelationID      string            `json:"correlation_id,omitempty"`
    Metadata           map[string]string `json:"metadata,omitempty"`
}
```

### 10. Subscription Item Creation Logic

When creating a subscription item, the service should copy pricing fields from the associated price to ensure billing stability:

```go
// internal/application/services/subscription_service.go
func (s *SubscriptionService) CreateSubscriptionItem(ctx context.Context, orgId string, input dto.CreateSubscriptionItemInput) (entities.SubscriptionItem, error) {
    // Get the price to copy pricing configuration
    price, err := s.priceService.Get(ctx, orgId, input.PriceId)
    if err != nil {
        return entities.SubscriptionItem{}, fmt.Errorf("price not found: %w", err)
    }
    
    // Create price snapshot for audit/comparison
    priceSnapshot, err := json.Marshal(price)
    if err != nil {
        return entities.SubscriptionItem{}, fmt.Errorf("failed to create price snapshot: %w", err)
    }
    
    // Create subscription item with copied pricing fields
    subscriptionItem := entities.SubscriptionItem{
        // ... basic fields ...
        PriceId:            input.PriceId,
        MeterId:            price.MeterId,
        
        // Copy pricing configuration (locked at creation)
        HasUsage:           price.HasUsage,
        PercentageRate:     price.PercentageRate,
        FixedFee:           price.FixedFee,
        UnitPrice:          price.UnitPrice,
        OverageUnitPrice:   price.OverageUnitPrice,
        IncludedUsage:      price.IncludedUsage,
        UsageLimit:         price.UsageLimit,
        
        // Store snapshot for comparison
        PriceSnapshot:      priceSnapshot,
        
        // ... other fields ...
    }
    
    return s.subscriptionItemRepo.Create(ctx, subscriptionItem)
}
```

## Summary of Changes

1. **New Meter Entity**: Separates measurement logic from pricing
2. **Updated Price Model**: References meters instead of embedding usage config
3. **Updated SubscriptionItem**: References meters, keeps pricing config
4. **New Meter Service**: Manages meter CRUD operations
5. **Updated Usage Recording**: Uses meter configuration for event creation
6. **Flexible Event Structure**: Supports various meter types and configurations
7. **API Updates**: New meter endpoints and updated price creation

This architecture provides clear separation between:
- **What to measure** (Meters)
- **How much to charge** (Prices)
- **What was used** (Usage Events)