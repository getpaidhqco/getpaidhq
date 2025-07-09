# Usage-Based Billing Architecture Implementation Specification

## Overview

This specification outlines the implementation of an event-driven PostgreSQL architecture for usage-based billing that separates high-volume usage recording from core business operations while maintaining data consistency through event sourcing and time-series optimized PostgreSQL features.

## Architecture Goals

1. **Future-Proof Scalability**: Built to handle growth from MVP (few transactions/day) to high volume
2. **Logical Separation**: Isolate usage data from core business operations
3. **Billing Accuracy**: Ensure eventual consistency with 30-minute settlement window
4. **Real-time Analytics**: Support near real-time usage dashboards
5. **Scalability**: Independent scaling of usage recording and billing systems

## System Architecture

### Core Components

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   API       │────▶│   Event     │────▶│  Usage DB    │────▶│   Main DB   │
│   (Usage)   │     │  Publisher  │     │(PostgreSQL)  │     │  (Billing)  │
└─────────────┘     └─────────────┘     └──────────────┘     └─────────────┘
                                              │                     ▲
                                              └─────────────────────┘
                                             2:30AM Billing Process
```

### Data Flow

1. **Usage Recording**: API → Event Publisher → PostgreSQL Usage DB (Real-time)
2. **Analytics**: PostgreSQL Materialized Views (5-minute refresh)
3. **Billing**: Scheduled job at 2:30 AM queries finalized aggregates
4. **Customer Dashboards**: Query materialized views for near real-time data

## Database Architecture

### PostgreSQL Usage Database

**Purpose**: High-volume usage event storage and time-series analytics with partitioning
**Port**: 5433 (or separate RDS instance)
**Database**: `payloop_usage`

### PostgreSQL Main Database

**Purpose**: Core business logic, subscriptions, invoices, customers
**Port**: 5432  
**Database**: `payloop`

### Event Publisher

**Purpose**: Event streaming and decoupling
**Interface**: `DurableEventPublisher`
**Implementation**: Can be Kafka, NATS, or any other event streaming system
**Topics**: `usage-events`, `usage-processed`

## Implementation Tasks

### 1. Database Schema Setup

#### A. Create Usage Database Prisma Schema

**File**: `schemas/usage/schema.prisma`

```prisma
generator client {
  provider = "prisma-client-js"
  output   = "../../node_modules/@prisma/usage-client"
}

datasource db {
  provider = "postgresql"
  url      = env("USAGE_DATABASE_URL")
}

model UsageEvent {
  time               DateTime @db.Timestamptz
  orgId              String   @map("org_id")
  subscriptionId     String   @map("subscription_id")
  subscriptionItemId String   @map("subscription_item_id")
  customerId         String   @map("customer_id")
  usageType          String   @map("usage_type")
  quantity           Decimal? @db.Decimal(15, 4)
  transactionValue   BigInt?  @map("transaction_value")
  calculatedAmount   BigInt   @map("calculated_amount")
  referenceId        String?  @map("reference_id")
  referenceType      String?  @map("reference_type")
  metadata           Json?
  
  @@id([time, orgId, subscriptionItemId])
  @@map("usage_events")
}
```

#### B. Remove UsageRecord from Main Schema

**File**: `prisma/schema.prisma`

Remove the following:

1. **UsageRecord model** (lines 660-721)
2. **Relationship declarations**:
   - Line 112: `UsageRecord UsageRecord[]` from Org model
   - Line 285: `UsageRecord UsageRecord[]` from Price model  
   - Line 465: `UsageRecord UsageRecord[]` from Customer model
   - Line 598: `UsageRecord UsageRecord[]` from Subscription model
   - Line 644: `usageRecords UsageRecord[]` from SubscriptionItem model
   - Line 956: `UsageRecord UsageRecord[]` from Invoice model

### 2. PostgreSQL Usage Database Setup

#### A. Create PostgreSQL Usage Database Schema

**Note**: The usage_events table will be created by Prisma migrations as a partitioned table. Time-based partitioning setup is handled by a separate script.

**Partition Setup Script**: `scripts/setup-usage-partitions.sql`

This script:
- Creates monthly partitions starting from July 2025
- Sets up automated partition management functions
- Configures partition cleanup and maintenance
- Provides monitoring and verification tools

**Key Features**:
- **Monthly Partitioning**: Each month gets its own partition (e.g., `usage_events_2025_07`)
- **Automatic Creation**: New partitions created 6 months in advance
- **Automatic Cleanup**: Old partitions (>5 years) automatically dropped
- **Optimized Indexes**: Each partition gets performance-optimized indexes
- **Monitoring**: Built-in functions to monitor partition health and size

#### B. Create Materialized Views for Aggregations

**File**: `scripts/create-materialized-views.sql`

```sql
-- Hourly usage aggregates for real-time dashboards
-- Using standard PostgreSQL materialized views for efficient querying
CREATE MATERIALIZED VIEW usage_hourly AS
SELECT 
    date_trunc('hour', time) AS hour,
    org_id,
    subscription_id,
    subscription_item_id,
    usage_type,
    SUM(quantity) as total_quantity,
    SUM(calculated_amount) as total_amount,
    COUNT(*) as event_count,
    MAX(time) as last_event_time
FROM usage_events
GROUP BY hour, org_id, subscription_id, subscription_item_id, usage_type;

-- Create index on the materialized view for fast queries
CREATE UNIQUE INDEX idx_usage_hourly_unique ON usage_hourly (hour, org_id, subscription_item_id, usage_type);
CREATE INDEX idx_usage_hourly_org_time ON usage_hourly (org_id, hour DESC);

-- Daily usage aggregates for billing
CREATE MATERIALIZED VIEW usage_daily_billing AS
SELECT 
    date_trunc('day', time) AS day,
    org_id,
    subscription_id,
    subscription_item_id,
    usage_type,
    date_trunc('month', time) as billing_period,
    SUM(quantity) as daily_quantity,
    SUM(calculated_amount) as daily_amount,
    COUNT(*) as daily_events,
    MIN(time) as first_event_time,
    MAX(time) as last_event_time
FROM usage_events
GROUP BY day, org_id, subscription_id, subscription_item_id, usage_type, billing_period;

-- Create index on daily billing view
CREATE UNIQUE INDEX idx_usage_daily_billing_unique ON usage_daily_billing (day, org_id, subscription_item_id, usage_type);
CREATE INDEX idx_usage_daily_billing_org_period ON usage_daily_billing (org_id, billing_period, subscription_item_id);

-- Monthly summary for analytics
CREATE MATERIALIZED VIEW usage_monthly_summary AS
SELECT 
    date_trunc('month', time) AS month,
    org_id,
    subscription_id,
    subscription_item_id,
    usage_type,
    SUM(quantity) as monthly_quantity,
    SUM(calculated_amount) as monthly_amount,
    AVG(quantity) as avg_quantity,
    MAX(quantity) as max_quantity,
    COUNT(DISTINCT DATE(time)) as active_days,
    COUNT(*) as total_events
FROM usage_events
GROUP BY month, org_id, subscription_id, subscription_item_id, usage_type;

-- Create index on monthly summary
CREATE UNIQUE INDEX idx_usage_monthly_summary_unique ON usage_monthly_summary (month, org_id, subscription_item_id, usage_type);
CREATE INDEX idx_usage_monthly_summary_org ON usage_monthly_summary (org_id, month DESC);

-- Function to refresh materialized views (called by scheduler)
CREATE OR REPLACE FUNCTION refresh_usage_aggregates()
RETURNS void AS $$
BEGIN
    -- Refresh materialized views concurrently (non-blocking)
    REFRESH MATERIALIZED VIEW CONCURRENTLY usage_hourly;
    REFRESH MATERIALIZED VIEW CONCURRENTLY usage_daily_billing;
    REFRESH MATERIALIZED VIEW CONCURRENTLY usage_monthly_summary;
    
    -- Log the refresh
    INSERT INTO usage_event_log (
        org_id,
        event_type,
        triggered_by,
        reason,
        metadata
    ) VALUES (
        'system',
        'materialized_views_refreshed',
        'scheduler',
        'Materialized views refreshed for real-time analytics',
        jsonb_build_object('refresh_time', NOW())
    );
END;
$$ LANGUAGE plpgsql;

-- Function to get billing summary from materialized views
CREATE OR REPLACE FUNCTION get_monthly_billing_summary(
    p_org_id TEXT,
    p_billing_period DATE
)
RETURNS TABLE (
    subscription_id TEXT,
    subscription_item_id TEXT,
    usage_type TEXT,
    total_quantity NUMERIC,
    total_amount BIGINT,
    daily_events BIGINT,
    active_days BIGINT,
    first_usage TIMESTAMPTZ,
    last_usage TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        udb.subscription_id,
        udb.subscription_item_id,
        udb.usage_type,
        SUM(udb.daily_quantity) as total_quantity,
        SUM(udb.daily_amount) as total_amount,
        SUM(udb.daily_events) as daily_events,
        COUNT(DISTINCT udb.day) as active_days,
        MIN(udb.first_event_time) as first_usage,
        MAX(udb.last_event_time) as last_usage
    FROM usage_daily_billing udb
    WHERE udb.org_id = p_org_id
      AND udb.billing_period = p_billing_period
    GROUP BY 
        udb.subscription_id, 
        udb.subscription_item_id, 
        udb.usage_type
    ORDER BY 
        udb.subscription_id, 
        udb.subscription_item_id;
END;
$$ LANGUAGE plpgsql;
```

### 3. Infrastructure Setup

#### A. Update Docker Compose

**File**: `docker/docker-compose.yml`

Add the following services:

```yaml
  # PostgreSQL for usage data  
  usage-postgres:
    image: postgres:15-alpine
    container_name: payloop-usage-db
    environment:
      POSTGRES_DB: payloop_usage
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_INITDB_ARGS: "-E UTF8"
    ports:
      - "5433:5432"
    volumes:
      - usage_postgres_data:/var/lib/postgresql/data
      - ./schemas/usage/migrations:/docker-entrypoint-initdb.d
    networks:
      - payloop-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d payloop_usage"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Kafka for event streaming
  kafka:
    image: bitnami/kafka:3.5
    container_name: payloop-kafka
    environment:
      - KAFKA_CFG_NODE_ID=0
      - KAFKA_CFG_PROCESS_ROLES=controller,broker
      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
      - KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true
    ports:
      - "9092:9092"
    volumes:
      - kafka_data:/bitnami/kafka
    networks:
      - payloop-network
    healthcheck:
      test: ["CMD-SHELL", "kafka-topics.sh --bootstrap-server localhost:9092 --list"]
      interval: 30s
      timeout: 10s
      retries: 3


volumes:
  usage_postgres_data:
  kafka_data:
```

### 4. Event Schema Definition

#### A. Usage Event Structure

**File**: `internal/domain/events/usage_events.go`

```go
package events

import (
    "time"
)

// UsageEvent represents a single usage event in the system
type UsageEvent struct {
    // Event metadata
    EventID       string    `json:"event_id"`
    EventType     string    `json:"event_type"` // "usage.recorded", "usage.processed", "usage.corrected"
    EventVersion  string    `json:"event_version"`
    Timestamp     time.Time `json:"timestamp"`
    
    // Business context
    OrgID              string `json:"org_id"`
    SubscriptionID     string `json:"subscription_id"`
    SubscriptionItemID string `json:"subscription_item_id"`
    CustomerID         string `json:"customer_id"`
    
    // Usage data
    UsageType        string  `json:"usage_type"`
    Quantity         float64 `json:"quantity,omitempty"`
    TransactionValue int64   `json:"transaction_value,omitempty"`
    CalculatedAmount int64   `json:"calculated_amount"`
    
    // References
    ReferenceID   string `json:"reference_id,omitempty"`
    ReferenceType string `json:"reference_type,omitempty"`
    
    // Event sourcing metadata
    CausationID   string            `json:"causation_id,omitempty"`
    CorrelationID string            `json:"correlation_id,omitempty"`
    Metadata      map[string]string `json:"metadata,omitempty"`
}

// UsageProcessedEvent indicates usage has been included in billing
type UsageProcessedEvent struct {
    EventID            string    `json:"event_id"`
    Timestamp          time.Time `json:"timestamp"`
    OrgID              string    `json:"org_id"`
    SubscriptionID     string    `json:"subscription_id"`
    SubscriptionItemID string    `json:"subscription_item_id"`
    BillingPeriod      string    `json:"billing_period"`
    InvoiceID          string    `json:"invoice_id"`
    TotalAmount        int64     `json:"total_amount"`
    EventCount         int       `json:"event_count"`
}
```

### 5. Service Implementation

#### A. Usage Recording Service

**File**: `internal/application/services/usage_recording_service.go`

```go
package services

import (
    "context"
    "fmt"
    "time"
    
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
    "payloop/internal/domain/events"
    "payloop/internal/infrastructure/messaging"
)

type UsageRecordingService struct {
    eventPublisher   events.DurableEventPublisher
    subscriptionRepo repositories.SubscriptionRepository
}

func (s *UsageRecordingService) RecordUsage(ctx context.Context, orgID string, input dto.RecordUsageInput) error {
    // 1. Validate subscription item exists and has usage enabled
    subscriptionItem, err := s.subscriptionRepo.GetSubscriptionItem(ctx, orgID, input.SubscriptionItemID)
    if err != nil {
        return fmt.Errorf("subscription item not found: %w", err)
    }
    
    if !subscriptionItem.HasUsage {
        return fmt.Errorf("subscription item %s does not support usage recording", input.SubscriptionItemID)
    }
    
    // 2. Calculate amount based on subscription item pricing configuration
    calculatedAmount, err := s.calculateUsageAmount(subscriptionItem, input)
    if err != nil {
        return fmt.Errorf("failed to calculate usage amount: %w", err)
    }
    
    // 3. Create usage event
    event := events.UsageEvent{
        EventID:            uuid.New().String(),
        EventType:          "usage.recorded",
        EventVersion:       "1.0",
        Timestamp:          time.Now(),
        OrgID:              orgID,
        SubscriptionID:     input.SubscriptionID,
        SubscriptionItemID: input.SubscriptionItemID,
        CustomerID:         input.CustomerID,
        UsageType:          subscriptionItem.UsageType,
        Quantity:           input.Quantity,
        TransactionValue:   input.TransactionValue,
        CalculatedAmount:   calculatedAmount,
        ReferenceID:        input.ReferenceID,
        ReferenceType:      input.ReferenceType,
        Metadata:           input.Metadata,
    }
    
    // 4. Publish to event stream
    return s.eventPublisher.PublishUsageEvent(ctx, event)
}

func (s *UsageRecordingService) calculateUsageAmount(item entities.SubscriptionItem, input dto.RecordUsageInput) (int64, error) {
    switch item.UsageType {
    case "unit":
        return int64(input.Quantity * float64(item.UnitPrice)), nil
    case "percentage":
        return int64(float64(input.TransactionValue) * item.PercentageRate / 100.0), nil
    case "hybrid":
        percentageFee := int64(float64(input.TransactionValue) * item.PercentageRate / 100.0)
        return percentageFee + item.FixedFee, nil
    default:
        return 0, fmt.Errorf("unsupported usage type: %s", item.UsageType)
    }
}
```

#### B. Usage Aggregation Service

**File**: `internal/application/services/usage_aggregation_service.go`

```go
package services

import (
    "context"
    "database/sql"
    "time"
    
    "payloop/internal/application/dto"
)

type UsageAggregationService struct {
    usageDB *sql.DB
}

// GetMonthlyUsage retrieves aggregated usage for billing period
func (s *UsageAggregationService) GetMonthlyUsage(ctx context.Context, orgID string, billingPeriod time.Time) ([]dto.MonthlyUsageAggregate, error) {
    query := `
        SELECT 
            subscription_id,
            subscription_item_id,
            usage_type,
            SUM(daily_quantity) as total_quantity,
            SUM(daily_amount) as total_amount,
            COUNT(DISTINCT day) as active_days,
            SUM(daily_events) as total_events,
            MIN(first_event_time) as period_start,
            MAX(last_event_time) as period_end
        FROM usage_daily_billing
        WHERE org_id = $1 
          AND billing_period = $2
        GROUP BY subscription_id, subscription_item_id, usage_type
        ORDER BY subscription_id, subscription_item_id
    `
    
    rows, err := s.usageDB.QueryContext(ctx, query, orgID, billingPeriod)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var aggregates []dto.MonthlyUsageAggregate
    for rows.Next() {
        var agg dto.MonthlyUsageAggregate
        err := rows.Scan(
            &agg.SubscriptionID,
            &agg.SubscriptionItemID,
            &agg.UsageType,
            &agg.TotalQuantity,
            &agg.TotalAmount,
            &agg.ActiveDays,
            &agg.TotalEvents,
            &agg.PeriodStart,
            &agg.PeriodEnd,
        )
        if err != nil {
            return nil, err
        }
        aggregates = append(aggregates, agg)
    }
    
    return aggregates, nil
}

// GetRealtimeUsage retrieves near real-time usage from continuous aggregates
func (s *UsageAggregationService) GetRealtimeUsage(ctx context.Context, orgID, subscriptionItemID string, since time.Time) (dto.UsageSummary, error) {
    query := `
        SELECT 
            COALESCE(SUM(total_quantity), 0) as quantity,
            COALESCE(SUM(total_amount), 0) as amount,
            COALESCE(SUM(event_count), 0) as events,
            MAX(last_event_time) as last_usage
        FROM usage_hourly
        WHERE org_id = $1 
          AND subscription_item_id = $2
          AND hour >= $3
    `
    
    var summary dto.UsageSummary
    err := s.usageDB.QueryRowContext(ctx, query, orgID, subscriptionItemID, since).Scan(
        &summary.Quantity,
        &summary.Amount,
        &summary.Events,
        &summary.LastUsage,
    )
    
    return summary, err
}
```

### 6. Billing Integration

#### A. Updated Billing Service

**File**: `internal/application/services/billing_service.go`

Update the billing service to use PostgreSQL aggregates:

```go
// Add usage aggregation dependency
type BillingService struct {
    // ... existing fields
    usageAggregation *UsageAggregationService
    eventPublisher    messaging.KafkaProducer
}

// ProcessMonthlyBilling integrates with usage aggregates
func (s *BillingService) ProcessMonthlyBilling(ctx context.Context, orgID string, billingPeriod time.Time) error {
    // 1. Get active subscriptions from main DB
    subscriptions, err := s.subscriptionRepo.GetActiveSubscriptions(ctx, orgID, billingPeriod)
    if err != nil {
        return err
    }

    // 2. Get usage aggregates from PostgreSQL Usage DB
    usageAggregates, err := s.usageAggregation.GetMonthlyUsage(ctx, orgID, billingPeriod)
    if err != nil {
        return err
    }

    // 3. Create usage lookup map
    usageMap := make(map[string]dto.MonthlyUsageAggregate)
    for _, usage := range usageAggregates {
        key := fmt.Sprintf("%s:%s", usage.SubscriptionID, usage.SubscriptionItemID)
        usageMap[key] = usage
    }

    // 4. Process each subscription
    for _, subscription := range subscriptions {
        invoice, err := s.createInvoiceWithUsage(ctx, subscription, usageMap)
        if err != nil {
            continue // Log error and continue with next subscription
        }

        // 5. Publish usage processed events
        s.publishUsageProcessedEvents(ctx, subscription, usageMap, invoice.ID, billingPeriod)
    }

    return nil
}

func (s *BillingService) createInvoiceWithUsage(ctx context.Context, subscription entities.Subscription, usageMap map[string]dto.MonthlyUsageAggregate) (entities.Invoice, error) {
    // Get subscription items
    items, err := s.subscriptionRepo.GetSubscriptionItems(ctx, subscription.OrgID, subscription.ID)
    if err != nil {
        return entities.Invoice{}, err
    }

    var invoiceItems []entities.InvoiceLineItem
    
    for _, item := range items {
        // Fixed recurring charges
        if item.Amount > 0 {
            invoiceItems = append(invoiceItems, entities.InvoiceLineItem{
                Description: item.Description,
                Quantity:    decimal.NewFromInt(int64(item.Quantity)),
                UnitPrice:   item.Amount,
                LineTotal:   item.Amount * item.Quantity,
                Category:    "recurring",
            })
        }

        // Usage-based charges
        if item.HasUsage {
            key := fmt.Sprintf("%s:%s", subscription.ID, item.ID)
            if usage, exists := usageMap[key]; exists && usage.TotalAmount > 0 {
                invoiceItems = append(invoiceItems, entities.InvoiceLineItem{
                    Description: fmt.Sprintf("%s - Usage (%s events)", item.Description, usage.TotalEvents),
                    Quantity:    decimal.NewFromFloat(usage.TotalQuantity),
                    UnitPrice:   0, // Usage already calculated
                    LineTotal:   int(usage.TotalAmount),
                    Category:    "usage",
                })
            }
        }
    }

    // Create invoice
    return s.CreateInvoice(ctx, subscription, invoiceItems)
}
```


### 7. Kafka Consumer Integration

#### A. Usage Service with Kafka Consumer Goroutine

**File**: `internal/application/services/usage_recording_service.go`

Update the existing usage recording service to include the Kafka consumer:

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/segmentio/kafka-go"
    "payloop/internal/application/dto"
    "payloop/internal/domain/entities"
    "payloop/internal/domain/events"
    "payloop/internal/domain/repositories"
    "payloop/internal/infrastructure/messaging"
)

type UsageRecordingService struct {
    eventPublisher        messaging.KafkaProducer
    kafkaConsumer        messaging.KafkaConsumer
    subscriptionRepo     repositories.SubscriptionRepository
    usageEventRepo       repositories.UsageEventRepository
    usageProcessingRepo  repositories.UsageProcessingStatusRepository
    logger               logger.Logger
    stopConsumer         chan struct{}
}

func NewUsageRecordingService(
    eventPublisher messaging.KafkaProducer,
    kafkaConsumer messaging.KafkaConsumer,
    subscriptionRepo repositories.SubscriptionRepository,
    usageEventRepo repositories.UsageEventRepository,
    usageProcessingRepo repositories.UsageProcessingStatusRepository,
    logger logger.Logger,
) *UsageRecordingService {
    return &UsageRecordingService{
        eventPublisher:       eventPublisher,
        kafkaConsumer:       kafkaConsumer,
        subscriptionRepo:    subscriptionRepo,
        usageEventRepo:      usageEventRepo,
        usageProcessingRepo: usageProcessingRepo,
        logger:              logger,
        stopConsumer:        make(chan struct{}),
    }
}

func (s *UsageRecordingService) Start(ctx context.Context) {
    // Start Kafka consumer as goroutine
    go s.consumeUsageEvents(ctx)
}

func (s *UsageRecordingService) Stop() {
    close(s.stopConsumer)
}

func (s *UsageRecordingService) RecordUsage(ctx context.Context, orgID string, input dto.RecordUsageInput) error {
    // 1. Validate subscription item exists and has usage enabled
    subscriptionItem, err := s.subscriptionRepo.GetSubscriptionItem(ctx, orgID, input.SubscriptionItemID)
    if err != nil {
        return fmt.Errorf("subscription item not found: %w", err)
    }
    
    if !subscriptionItem.HasUsage {
        return fmt.Errorf("subscription item %s does not support usage recording", input.SubscriptionItemID)
    }
    
    // 2. Calculate amount based on subscription item pricing configuration
    calculatedAmount, err := s.calculateUsageAmount(subscriptionItem, input)
    if err != nil {
        return fmt.Errorf("failed to calculate usage amount: %w", err)
    }
    
    // 3. Create usage event
    event := events.UsageEvent{
        EventID:            uuid.New().String(),
        EventType:          "usage.recorded",
        EventVersion:       "1.0",
        Timestamp:          time.Now(),
        OrgID:              orgID,
        SubscriptionID:     input.SubscriptionID,
        SubscriptionItemID: input.SubscriptionItemID,
        CustomerID:         input.CustomerID,
        UsageType:          subscriptionItem.UsageType,
        Quantity:           input.Quantity,
        TransactionValue:   input.TransactionValue,
        CalculatedAmount:   calculatedAmount,
        ReferenceID:        input.ReferenceID,
        ReferenceType:      input.ReferenceType,
        Metadata:           input.Metadata,
    }
    
    // 4. Publish to event stream
    return s.eventPublisher.PublishUsageEvent(ctx, event)
}

func (s *UsageRecordingService) consumeUsageEvents(ctx context.Context) {
    s.logger.Info("Starting Kafka consumer for usage events")
    
    for {
        select {
        case <-ctx.Done():
            s.logger.Info("Context cancelled, stopping usage event consumer")
            return
        case <-s.stopConsumer:
            s.logger.Info("Stop signal received, stopping usage event consumer")
            return
        default:
            msg, err := s.kafkaConsumer.ReadMessage(ctx)
            if err != nil {
                s.logger.Error("Error reading Kafka message", "error", err)
                continue
            }
            
            if err := s.processUsageEvent(ctx, msg); err != nil {
                s.logger.Error("Error processing usage event", "error", err, "message", string(msg.Value))
            }
        }
    }
}

func (s *UsageRecordingService) processUsageEvent(ctx context.Context, msg kafka.Message) error {
    var event events.UsageEvent
    if err := json.Unmarshal(msg.Value, &event); err != nil {
        return fmt.Errorf("failed to unmarshal usage event: %w", err)
    }
    
    // Convert event to entity
    usageEvent := entities.UsageEvent{
        Time:               event.Timestamp,
        OrgID:              event.OrgID,
        SubscriptionID:     event.SubscriptionID,
        SubscriptionItemID: event.SubscriptionItemID,
        CustomerID:         event.CustomerID,
        UsageType:          event.UsageType,
        Quantity:           &event.Quantity,
        TransactionValue:   &event.TransactionValue,
        CalculatedAmount:   event.CalculatedAmount,
        ReferenceID:        &event.ReferenceID,
        ReferenceType:      &event.ReferenceType,
        Metadata:           event.Metadata,
    }
    
    // Insert into PostgreSQL Usage DB
    if err := s.usageEventRepo.Create(ctx, usageEvent); err != nil {
        return fmt.Errorf("failed to insert usage event: %w", err)
    }
    
    s.logger.Debug("Usage event processed successfully", 
        "event_id", event.EventID,
        "org_id", event.OrgID,
        "subscription_item_id", event.SubscriptionItemID)
    
    return nil
}

func (s *UsageRecordingService) calculateUsageAmount(item entities.SubscriptionItem, input dto.RecordUsageInput) (int64, error) {
    switch item.UsageType {
    case "unit":
        return int64(input.Quantity * float64(item.UnitPrice)), nil
    case "percentage":
        return int64(float64(input.TransactionValue) * item.PercentageRate / 100.0), nil
    case "hybrid":
        percentageFee := int64(float64(input.TransactionValue) * item.PercentageRate / 100.0)
        return percentageFee + item.FixedFee, nil
    default:
        return 0, fmt.Errorf("unsupported usage type: %s", item.UsageType)
    }
}
```

### 8. PostgreSQL Usage Repository Implementation

#### A. Usage Event Repository Interface

**File**: `internal/domain/repositories/usage_event.go`

```go
package repositories

import (
    "context"
    "time"
    
    "payloop/internal/domain/entities"
)

type UsageEventRepository interface {
    // Create inserts a new usage event
    Create(ctx context.Context, event entities.UsageEvent) error
    
    // BatchCreate inserts multiple usage events efficiently
    BatchCreate(ctx context.Context, events []entities.UsageEvent) error
    
    // FindByID retrieves a usage event by composite key
    FindByID(ctx context.Context, orgID, subscriptionItemID string, time time.Time) (entities.UsageEvent, error)
    
    // FindBySubscriptionItem retrieves usage events for a subscription item
    FindBySubscriptionItem(ctx context.Context, orgID, subscriptionItemID string, 
        startTime, endTime time.Time) ([]entities.UsageEvent, error)
    
    // FindByReferenceID retrieves usage event by reference (for idempotency)
    FindByReferenceID(ctx context.Context, referenceID, referenceType string) (entities.UsageEvent, error)
    
    // Delete removes a usage event (for corrections)
    Delete(ctx context.Context, orgID, subscriptionItemID string, time time.Time) error
}

type UsageProcessingStatusRepository interface {
    // Create or update processing status for a billing period
    UpsertProcessingStatus(ctx context.Context, status entities.UsageProcessingStatus) error
    
    // Get processing status for a billing period
    GetProcessingStatus(ctx context.Context, orgID, subscriptionItemID, billingPeriod string) (entities.UsageProcessingStatus, error)
    
    // Get all unprocessed usage for billing
    GetUnprocessedUsage(ctx context.Context, orgID, billingPeriod string) ([]entities.UsageProcessingStatus, error)
    
    // Mark usage as processed with invoice ID
    MarkAsProcessed(ctx context.Context, orgID, subscriptionItemID, billingPeriod, invoiceID string) error
    
    // Get processing status by invoice
    GetByInvoiceID(ctx context.Context, invoiceID string) ([]entities.UsageProcessingStatus, error)
}

type UsageAggregationRepository interface {
    // Get monthly usage aggregates for billing
    GetMonthlyUsage(ctx context.Context, orgID string, billingPeriod time.Time) ([]entities.MonthlyUsageAggregate, error)
    
    // Get real-time usage summary from materialized views
    GetRealtimeUsage(ctx context.Context, orgID, subscriptionItemID string, 
        since time.Time) (entities.UsageSummary, error)
    
    // Get customer usage summary
    GetCustomerUsage(ctx context.Context, orgID, customerID string, 
        startTime, endTime time.Time) (entities.CustomerUsageSummary, error)
    
    // Get usage analytics by type
    GetUsageTypeAnalytics(ctx context.Context, orgID string, 
        startTime, endTime time.Time) ([]entities.UsageTypeAnalytics, error)
    
    // Refresh materialized views manually (for billing consistency)
    RefreshAggregates(ctx context.Context) error
}
```

#### B. PostgreSQL Usage Repository Implementation

**File**: `internal/infrastructure/db/postgres/usage_event_repository.go`

```go
package postgres

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/jackc/pgx/v5"
    "payloop/internal/application/lib/logger"
    "payloop/internal/domain/entities"
    "payloop/internal/domain/repositories"
    "payloop/internal/lib"
)

type UsageEventRepository struct {
    *PgDatabase
    logger logger.Logger
}

func NewUsageEventRepository(usageDb lib.Database, logger logger.Logger) repositories.UsageEventRepository {
    pgDatabase, ok := usageDb.(*PgDatabase)
    if !ok {
        panic("database is not of type *PgDatabase")
    }
    return &UsageEventRepository{
        PgDatabase: pgDatabase,
        logger:     logger,
    }
}

func (r *UsageEventRepository) Create(ctx context.Context, event entities.UsageEvent) error {
    tx := r.getTransactionFromContext(ctx)
    
    metadataJSON, err := json.Marshal(event.Metadata)
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %w", err)
    }
    
    query := `
        INSERT INTO usage_events (
            time, org_id, subscription_id, subscription_item_id, customer_id,
            usage_type, quantity, transaction_value, calculated_amount,
            reference_id, reference_type, metadata
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT (org_id, subscription_item_id, time) DO NOTHING
    `
    
    _, err = tx.Exec(ctx, query,
        event.Time,
        event.OrgID,
        event.SubscriptionID,
        event.SubscriptionItemID,
        event.CustomerID,
        event.UsageType,
        event.Quantity,
        event.TransactionValue,
        event.CalculatedAmount,
        event.ReferenceID,
        event.ReferenceType,
        metadataJSON,
    )
    
    if err != nil {
        return fmt.Errorf("failed to insert usage event: %w", err)
    }
    
    return nil
}

func (r *UsageEventRepository) BatchCreate(ctx context.Context, events []entities.UsageEvent) error {
    if len(events) == 0 {
        return nil
    }
    
    tx := r.getTransactionFromContext(ctx)
    
    // Prepare batch insert
    batch := &pgx.Batch{}
    query := `
        INSERT INTO usage_events (
            time, org_id, subscription_id, subscription_item_id, customer_id,
            usage_type, quantity, transaction_value, calculated_amount,
            reference_id, reference_type, metadata
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT (org_id, subscription_item_id, time) DO NOTHING
    `
    
    for _, event := range events {
        metadataJSON, _ := json.Marshal(event.Metadata)
        batch.Queue(query,
            event.Time,
            event.OrgID,
            event.SubscriptionID,
            event.SubscriptionItemID,
            event.CustomerID,
            event.UsageType,
            event.Quantity,
            event.TransactionValue,
            event.CalculatedAmount,
            event.ReferenceID,
            event.ReferenceType,
            metadataJSON,
        )
    }
    
    results := tx.SendBatch(ctx, batch)
    defer results.Close()
    
    // Process batch results
    for i := 0; i < len(events); i++ {
        _, err := results.Exec()
        if err != nil {
            return fmt.Errorf("failed to insert batch event %d: %w", i, err)
        }
    }
    
    return nil
}

func (r *UsageEventRepository) FindByID(ctx context.Context, orgID, subscriptionItemID string, eventTime time.Time) (entities.UsageEvent, error) {
    tx := r.getTransactionFromContext(ctx)
    
    query := `
        SELECT time, org_id, subscription_id, subscription_item_id, customer_id,
               usage_type, quantity, transaction_value, calculated_amount,
               reference_id, reference_type, metadata
        FROM usage_events
        WHERE org_id = $1 AND subscription_item_id = $2 AND time = $3
    `
    
    var event entities.UsageEvent
    var metadataJSON []byte
    
    err := tx.QueryRow(ctx, query, orgID, subscriptionItemID, eventTime).Scan(
        &event.Time,
        &event.OrgID,
        &event.SubscriptionID,
        &event.SubscriptionItemID,
        &event.CustomerID,
        &event.UsageType,
        &event.Quantity,
        &event.TransactionValue,
        &event.CalculatedAmount,
        &event.ReferenceID,
        &event.ReferenceType,
        &metadataJSON,
    )
    
    if err != nil {
        if err == pgx.ErrNoRows {
            return entities.UsageEvent{}, fmt.Errorf("usage event not found")
        }
        return entities.UsageEvent{}, fmt.Errorf("failed to find usage event: %w", err)
    }
    
    if len(metadataJSON) > 0 {
        if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
            r.logger.Warn("Failed to unmarshal metadata", "error", err)
        }
    }
    
    return event, nil
}

func (r *UsageEventRepository) FindBySubscriptionItem(ctx context.Context, orgID, subscriptionItemID string, startTime, endTime time.Time) ([]entities.UsageEvent, error) {
    tx := r.getTransactionFromContext(ctx)
    
    query := `
        SELECT time, org_id, subscription_id, subscription_item_id, customer_id,
               usage_type, quantity, transaction_value, calculated_amount,
               reference_id, reference_type, metadata
        FROM usage_events
        WHERE org_id = $1 AND subscription_item_id = $2
          AND time >= $3 AND time < $4
        ORDER BY time DESC
    `
    
    rows, err := tx.Query(ctx, query, orgID, subscriptionItemID, startTime, endTime)
    if err != nil {
        return nil, fmt.Errorf("failed to query usage events: %w", err)
    }
    defer rows.Close()
    
    var events []entities.UsageEvent
    for rows.Next() {
        var event entities.UsageEvent
        var metadataJSON []byte
        
        err := rows.Scan(
            &event.Time,
            &event.OrgID,
            &event.SubscriptionID,
            &event.SubscriptionItemID,
            &event.CustomerID,
            &event.UsageType,
            &event.Quantity,
            &event.TransactionValue,
            &event.CalculatedAmount,
            &event.ReferenceID,
            &event.ReferenceType,
            &metadataJSON,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan usage event: %w", err)
        }
        
        if len(metadataJSON) > 0 {
            if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
                r.logger.Warn("Failed to unmarshal metadata", "error", err)
            }
        }
        
        events = append(events, event)
    }
    
    return events, nil
}

func (r *UsageEventRepository) FindByReferenceID(ctx context.Context, referenceID, referenceType string) (entities.UsageEvent, error) {
    tx := r.getTransactionFromContext(ctx)
    
    query := `
        SELECT time, org_id, subscription_id, subscription_item_id, customer_id,
               usage_type, quantity, transaction_value, calculated_amount,
               reference_id, reference_type, metadata
        FROM usage_events
        WHERE reference_id = $1 AND reference_type = $2
        LIMIT 1
    `
    
    var event entities.UsageEvent
    var metadataJSON []byte
    
    err := tx.QueryRow(ctx, query, referenceID, referenceType).Scan(
        &event.Time,
        &event.OrgID,
        &event.SubscriptionID,
        &event.SubscriptionItemID,
        &event.CustomerID,
        &event.UsageType,
        &event.Quantity,
        &event.TransactionValue,
        &event.CalculatedAmount,
        &event.ReferenceID,
        &event.ReferenceType,
        &metadataJSON,
    )
    
    if err != nil {
        if err == pgx.ErrNoRows {
            return entities.UsageEvent{}, fmt.Errorf("usage event not found")
        }
        return entities.UsageEvent{}, fmt.Errorf("failed to find usage event by reference: %w", err)
    }
    
    if len(metadataJSON) > 0 {
        if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
            r.logger.Warn("Failed to unmarshal metadata", "error", err)
        }
    }
    
    return event, nil
}

func (r *UsageEventRepository) Delete(ctx context.Context, orgID, subscriptionItemID string, eventTime time.Time) error {
    tx := r.getTransactionFromContext(ctx)
    
    query := `
        DELETE FROM usage_events
        WHERE org_id = $1 AND subscription_item_id = $2 AND time = $3
    `
    
    result, err := tx.Exec(ctx, query, orgID, subscriptionItemID, eventTime)
    if err != nil {
        return fmt.Errorf("failed to delete usage event: %w", err)
    }
    
    if result.RowsAffected() == 0 {
        return fmt.Errorf("usage event not found for deletion")
    }
    
    return nil
}
```

#### C. Usage Aggregation Service

**File**: `internal/application/services/usage_aggregation_service.go`

```go
package services

import (
    "context"
    "fmt"
    "time"
    
    "payloop/internal/application/dto"
    "payloop/internal/domain/repositories"
)

type UsageAggregationService struct {
    usageDB repositories.UsageAggregationRepository
    logger  logger.Logger
}

func NewUsageAggregationService(
    usageDB repositories.UsageAggregationRepository,
    logger logger.Logger,
) *UsageAggregationService {
    return &UsageAggregationService{
        usageDB: usageDB,
        logger:  logger,
    }
}

// GetMonthlyUsage retrieves aggregated usage for billing period from materialized views
func (s *UsageAggregationService) GetMonthlyUsage(ctx context.Context, orgID string, billingPeriod time.Time) ([]dto.MonthlyUsageAggregate, error) {
    s.logger.Debug("Getting monthly usage aggregates", "org_id", orgID, "period", billingPeriod)
    
    aggregates, err := s.usageDB.GetMonthlyUsage(ctx, orgID, billingPeriod)
    if err != nil {
        return nil, fmt.Errorf("failed to get monthly usage: %w", err)
    }
    
    s.logger.Info("Retrieved monthly usage aggregates", 
        "org_id", orgID, 
        "period", billingPeriod,
        "aggregate_count", len(aggregates))
    
    return aggregates, nil
}

// GetRealtimeUsage retrieves near real-time usage from materialized views
func (s *UsageAggregationService) GetRealtimeUsage(ctx context.Context, orgID, subscriptionItemID string, since time.Time) (dto.UsageSummary, error) {
    summary, err := s.usageDB.GetRealtimeUsage(ctx, orgID, subscriptionItemID, since)
    if err != nil {
        return dto.UsageSummary{}, fmt.Errorf("failed to get realtime usage: %w", err)
    }
    
    return summary, nil
}

// RefreshMaterializedViews manually refreshes all materialized views
func (s *UsageAggregationService) RefreshMaterializedViews(ctx context.Context) error {
    s.logger.Info("Refreshing materialized views for usage analytics")
    
    if err := s.usageDB.RefreshAggregates(ctx); err != nil {
        return fmt.Errorf("failed to refresh aggregates: %w", err)
    }
    
    s.logger.Info("Successfully refreshed materialized views")
    return nil
}
```

### 9. Environment Configuration and Setup

#### A. Environment Variables

Add to `.env`:

```env
# Usage Database
USAGE_DATABASE_URL=postgres://postgres:postgres@localhost:5433/payloop_usage

# Kafka
KAFKA_BROKERS=localhost:9092

# Usage Configuration
USAGE_PARTITION_RETENTION_MONTHS=60  # 5 years
USAGE_MATERIALIZED_VIEW_REFRESH_INTERVAL=5  # minutes
```

#### B. Database Setup Scripts

**File**: `scripts/setup-usage-db.sh`

```bash
#!/bin/bash
set -e

echo "Setting up PostgreSQL usage database with time-based partitioning..."

# Step 1: Generate Prisma client for usage database
echo "1. Generating Prisma client for usage database..."
pnpm dlx prisma generate --schema=schemas/usage/schema.prisma

# Step 2: Push schema to usage database (creates base tables)
echo "2. Pushing schema to PostgreSQL Usage DB..."
pnpm dlx prisma db push --schema=schemas/usage/schema.prisma

# Step 3: Set up time-based partitioning (July 2025 onwards)
echo "3. Setting up time-based partitioning..."
psql $USAGE_DATABASE_URL -f scripts/setup-usage-partitions.sql

# Step 4: Create materialized views for analytics
echo "4. Creating materialized views..."
psql $USAGE_DATABASE_URL -f scripts/create-materialized-views.sql

# Step 5: Verify setup
echo "5. Verifying partition setup..."
psql $USAGE_DATABASE_URL -c "SELECT * FROM get_partition_info('usage_events');"

echo "\n✅ Usage database setup completed!"
echo "📊 Partitions created: July 2025 - January 2026"
echo "📈 Materialized views ready for analytics"
echo "🔄 Automated maintenance configured"
```

#### C. Manual Operations

**Check Partition Status:**
```sql
SELECT * FROM get_partition_info('usage_events');
```

**Manual Partition Maintenance:**
```sql
SELECT maintain_usage_partitions();
```

**Refresh Materialized Views:**
```sql
SELECT refresh_usage_aggregates();
```

**Monitor Performance:**
```sql
SELECT * FROM get_materialized_view_stats();
```

### 9. Testing Strategy

#### A. Integration Tests

**File**: `internal/application/services/usage_integration_test.go`

```go
package services_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "payloop/internal/application/dto"
)

func TestUsageRecordingIntegration(t *testing.T) {
    // Setup test environment with PostgreSQL Usage DB and Kafka
    ctx := context.Background()
    
    t.Run("Record and Aggregate Usage", func(t *testing.T) {
        // 1. Record usage events
        for i := 0; i < 100; i++ {
            err := usageService.RecordUsage(ctx, "org1", dto.RecordUsageInput{
                SubscriptionID:     "sub1",
                SubscriptionItemID: "item1", 
                CustomerID:         "cust1",
                Quantity:           10.0,
                ReferenceID:        fmt.Sprintf("ref-%d", i),
            })
            assert.NoError(t, err)
        }
        
        // 2. Wait for events to be processed
        time.Sleep(10 * time.Second)
        
        // 3. Verify aggregation
        summary, err := usageAggregationService.GetRealtimeUsage(ctx, "org1", "item1", time.Now().Add(-1*time.Hour))
        assert.NoError(t, err)
        assert.Equal(t, float64(1000), summary.Quantity)
        assert.Equal(t, 100, summary.Events)
    })
}
```

### 10. Monitoring and Observability

#### A. Metrics Collection

Implement metrics for:
- Kafka message throughput
- PostgreSQL query performance
- Usage event processing duration
- Usage event lag time
- Materialized view refresh duration

#### B. Health Checks

**File**: `internal/infrastructure/health/usage_health.go`

```go
package health

import (
    "context"
    "database/sql"
    "time"
)

func CheckUsageDBHealth(ctx context.Context, db *sql.DB) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    _, err := db.ExecContext(ctx, "SELECT 1")
    return err
}

func CheckKafkaHealth(ctx context.Context, brokers []string) error {
    // Implement Kafka health check
    return nil
}
```

## Implementation Order

1. **Phase 1: Infrastructure Setup**
   - Update docker-compose.yml with PostgreSQL Usage DB and Kafka
   - Create usage Prisma schema and run `prisma db push`
   - Run partition setup script: `scripts/setup-usage-partitions.sql`
   - Create materialized views: `scripts/create-materialized-views.sql`
   - Remove UsageRecord from main schema

2. **Phase 2: Repository and Domain Layer**
   - Implement PostgreSQL Usage DB connection
   - Create usage event repository interfaces
   - Implement PostgreSQL repository implementations
   - Create usage domain entities

3. **Phase 3: Event System and Service Integration** 
   - Implement Kafka producer/consumer infrastructure
   - Create usage event definitions
   - Update usage recording service with Kafka producer
   - Add Kafka consumer goroutine to usage service
   - Create usage aggregation service

4. **Phase 4: Usage Data Access**
   - Implement usage aggregation service interfaces for external access
   - Create materialized views for efficient data retrieval
   - Set up automated materialized view refresh schedule (external scheduling required for RDS)

5. **Phase 5: Testing & Deployment**
   - Integration tests with PostgreSQL Usage DB and Kafka
   - API testing for usage recording
   - Usage aggregation accuracy testing
   - Production deployment and monitoring

## PostgreSQL Time-Based Partitioning Implementation

### Partition Strategy (July 2025 onwards)

**Monthly Range Partitioning:**
- `usage_events_2025_07` (July 1, 2025 - August 1, 2025)
- `usage_events_2025_08` (August 1, 2025 - September 1, 2025)
- `usage_events_2025_09` (September 1, 2025 - October 1, 2025)
- `usage_events_2025_10` (October 1, 2025 - November 1, 2025)
- `usage_events_2025_11` (November 1, 2025 - December 1, 2025)
- `usage_events_2025_12` (December 1, 2025 - January 1, 2026)
- `usage_events_2026_01` (January 1, 2026 - February 1, 2026)

**Automated Management:**
- **Future Partitions**: Created 6 months in advance
- **Cleanup**: Partitions older than 5 years automatically dropped
- **Maintenance**: Application-level scheduler or AWS Lambda (pg_cron not available on RDS)
- **Monitoring**: Built-in functions for partition health checks

**Performance Optimizations:**
- **Partition Pruning**: Queries automatically scan only relevant partitions
- **Parallel Processing**: Different partitions processed simultaneously
- **Optimized Indexes**: Each partition gets its own performance-tuned indexes
- **Materialized Views**: 5 views for efficient real-time analytics


## Success Criteria

- ✅ Handle current MVP volume (few transactions/day) with room for growth
- ✅ Real-time usage dashboards (< 5 minute latency via materialized views)
- ✅ Accurate usage aggregation with data consistency guarantees
- ✅ Zero data loss during processing (Kafka reliability)
- ✅ Automatic data compression and retention (PostgreSQL partitioning)
- ✅ Event-driven architecture ready for future scaling
- ✅ Clean separation of usage data from core business operations
- ✅ Simple deployment and monitoring for MVP
- ✅ **Time-based partitioning starting July 2025 with auto-management**
- ✅ **Zero TimescaleDB dependencies - pure PostgreSQL solution**

