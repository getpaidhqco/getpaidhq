# Usage-Based Billing Architecture Implementation Specification

## Overview

This specification outlines the implementation of a Kafka + TimescaleDB architecture for usage-based billing that separates high-volume usage recording from core business operations while maintaining data consistency through event sourcing and continuous aggregations.

## Architecture Goals

1. **High Throughput**: Handle millions of usage events per second
2. **Logical Separation**: Isolate usage data from core business operations
3. **Billing Accuracy**: Ensure eventual consistency with 2-hour settlement window
4. **Real-time Analytics**: Support near real-time usage dashboards
5. **Scalability**: Independent scaling of usage recording and billing systems

## System Architecture

### Core Components

```
┌─────────────┐     ┌─────────┐     ┌──────────────┐     ┌─────────────┐
│   API       │────▶│  Kafka  │────▶│ TimescaleDB  │────▶│   Main DB   │
│   (Usage)   │     │ (Events)│     │(Aggregates)  │     │  (Billing)  │
└─────────────┘     └─────────┘     └──────────────┘     └─────────────┘
                                            │                     ▲
                                            └─────────────────────┘
                                              2AM Billing Process
```

### Data Flow

1. **Usage Recording**: API → Kafka → TimescaleDB (Real-time)
2. **Analytics**: TimescaleDB Continuous Aggregates (5-minute refresh)
3. **Billing**: Scheduled job at 2 AM queries finalized aggregates
4. **Customer Dashboards**: Query continuous aggregates for near real-time data

## Database Architecture

### TimescaleDB (Usage Database)

**Purpose**: High-volume usage event storage and time-series analytics
**Port**: 5433
**Database**: `gphq_usage`

### PostgreSQL (Main Database)

**Purpose**: Core business logic, subscriptions, invoices, customers
**Port**: 5432  
**Database**: `gphq`

### Kafka

**Purpose**: Event streaming and decoupling
**Port**: 9092
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
  
  @@id([orgId, subscriptionItemId, time])
  @@index([time])
  @@index([orgId, subscriptionId])
  @@index([orgId, time])
  @@index([subscriptionItemId, time])
  @@index([referenceId, referenceType])
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

### 2. TimescaleDB Setup

#### A. Create TimescaleDB Initialization Script

**File**: `schemas/usage/migrations/001_initialize_timescaledb.sql`

```sql
-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Create usage_events table
CREATE TABLE usage_events (
    time TIMESTAMPTZ NOT NULL,
    org_id TEXT NOT NULL,
    subscription_id TEXT NOT NULL,
    subscription_item_id TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    usage_type TEXT NOT NULL,
    quantity NUMERIC(15, 4),
    transaction_value BIGINT,
    calculated_amount BIGINT NOT NULL,
    reference_id TEXT,
    reference_type TEXT,
    metadata JSONB,
    PRIMARY KEY (org_id, subscription_item_id, time)
);

-- Convert to hypertable (time-series optimization)
SELECT create_hypertable('usage_events', 'time', 
    chunk_time_interval => INTERVAL '1 day',
    create_default_indexes => FALSE
);

-- Create indexes
CREATE INDEX idx_usage_events_time ON usage_events (time DESC);
CREATE INDEX idx_usage_events_org_time ON usage_events (org_id, time DESC);
CREATE INDEX idx_usage_events_subscription ON usage_events (org_id, subscription_id);
CREATE INDEX idx_usage_events_subscription_item ON usage_events (subscription_item_id, time DESC);
CREATE INDEX idx_usage_events_reference ON usage_events (reference_id, reference_type) WHERE reference_id IS NOT NULL;

-- Add compression policy (compress data older than 7 days)
ALTER TABLE usage_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, subscription_id',
    timescaledb.compress_orderby = 'time DESC'
);

SELECT add_compression_policy('usage_events', INTERVAL '7 days');

-- Add retention policy (delete data older than 5 years)
SELECT add_retention_policy('usage_events', INTERVAL '5 years');
```

#### B. Create Continuous Aggregates

**File**: `schemas/usage/migrations/002_continuous_aggregates.sql`

```sql
-- Hourly usage aggregates for real-time dashboards
CREATE MATERIALIZED VIEW usage_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
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

-- Daily usage aggregates for billing
CREATE MATERIALIZED VIEW usage_daily_billing
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS day,
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

-- Monthly summary for analytics
CREATE MATERIALIZED VIEW usage_monthly_summary
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 month', time) AS month,
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

-- Add refresh policies
SELECT add_continuous_aggregate_policy('usage_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '5 minutes',
    schedule_interval => INTERVAL '5 minutes');

SELECT add_continuous_aggregate_policy('usage_daily_billing',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

SELECT add_continuous_aggregate_policy('usage_monthly_summary',
    start_offset => INTERVAL '3 months',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day');

-- Add compression for continuous aggregates
ALTER MATERIALIZED VIEW usage_hourly SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, subscription_id'
);

ALTER MATERIALIZED VIEW usage_daily_billing SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, subscription_id'
);

SELECT add_compression_policy('usage_hourly', INTERVAL '30 days');
SELECT add_compression_policy('usage_daily_billing', INTERVAL '90 days');
```

### 3. Infrastructure Setup

#### A. Update Docker Compose

**File**: `docker/docker-compose.yml`

Add the following services:

```yaml
  # TimescaleDB for usage data
  timescaledb:
    image: timescale/timescaledb:latest-pg15
    container_name: gphq-usage-db
    environment:
      POSTGRES_DB: gphq_usage
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_INITDB_ARGS: "-E UTF8"
    ports:
      - "5433:5432"
    volumes:
      - timescale_data:/var/lib/postgresql/data
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
  timescale_data:
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
    kafkaProducer     messaging.KafkaProducer
    subscriptionRepo  repositories.SubscriptionRepository
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
    
    // 4. Publish to Kafka
    return s.kafkaProducer.PublishUsageEvent(ctx, event)
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
    timescaleDB *sql.DB
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
    
    rows, err := s.timescaleDB.QueryContext(ctx, query, orgID, billingPeriod)
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
    err := s.timescaleDB.QueryRowContext(ctx, query, orgID, subscriptionItemID, since).Scan(
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

Update the billing service to use TimescaleDB aggregates:

```go
// Add usage aggregation dependency
type BillingService struct {
    // ... existing fields
    usageAggregation *UsageAggregationService
    kafkaProducer    messaging.KafkaProducer
}

// ProcessMonthlyBilling integrates with usage aggregates
func (s *BillingService) ProcessMonthlyBilling(ctx context.Context, orgID string, billingPeriod time.Time) error {
    // 1. Get active subscriptions from main DB
    subscriptions, err := s.subscriptionRepo.GetActiveSubscriptions(ctx, orgID, billingPeriod)
    if err != nil {
        return err
    }

    // 2. Get usage aggregates from TimescaleDB
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

#### B. Simple Scheduled Billing

**File**: `internal/application/services/billing_scheduler.go`

```go
package services

import (
    "context"
    "fmt"
    "time"
    
    "github.com/robfig/cron/v3"
    "payloop/internal/application/dto"
)

type BillingScheduler struct {
    billingService        BillingService
    usageAggregationSvc   UsageAggregationService
    orgRepository         repositories.OrgRepository
    cron                  *cron.Cron
}

func NewBillingScheduler(
    billingService BillingService,
    usageAggregationSvc UsageAggregationService,
    orgRepository repositories.OrgRepository,
) *BillingScheduler {
    return &BillingScheduler{
        billingService:      billingService,
        usageAggregationSvc: usageAggregationSvc,
        orgRepository:      orgRepository,
        cron:               cron.New(),
    }
}

func (s *BillingScheduler) Start() {
    // Schedule monthly billing on 1st of each month at 2:30 AM
    // 30-minute settlement window after midnight
    s.cron.AddFunc("30 2 1 * *", func() {
        s.processMonthlyBilling()
    })
    s.cron.Start()
}

func (s *BillingScheduler) processMonthlyBilling() {
    ctx := context.Background()
    
    // Get previous month's billing period
    now := time.Now()
    billingPeriod := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
    
    // Get all active organizations
    orgs, err := s.orgRepository.GetActiveOrganizations(ctx)
    if err != nil {
        log.Printf("Failed to get organizations for billing: %v", err)
        return
    }
    
    // Process billing for each organization
    for _, org := range orgs {
        if err := s.processBillingForOrg(ctx, org.ID, billingPeriod); err != nil {
            log.Printf("Failed to process billing for org %s: %v", org.ID, err)
            continue
        }
    }
}

func (s *BillingScheduler) processBillingForOrg(ctx context.Context, orgID string, billingPeriod time.Time) error {
    // Get usage aggregates from TimescaleDB
    usageAggregates, err := s.usageAggregationSvc.GetMonthlyUsage(ctx, orgID, billingPeriod)
    if err != nil {
        return fmt.Errorf("failed to get usage aggregates: %w", err)
    }
    
    // Process billing with usage data
    return s.billingService.ProcessMonthlyBilling(ctx, orgID, billingPeriod, usageAggregates)
}

func (s *BillingScheduler) Stop() {
    s.cron.Stop()
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
    kafkaProducer        messaging.KafkaProducer
    kafkaConsumer        messaging.KafkaConsumer
    subscriptionRepo     repositories.SubscriptionRepository
    usageEventRepo       repositories.UsageEventRepository
    usageProcessingRepo  repositories.UsageProcessingStatusRepository
    logger               logger.Logger
    stopConsumer         chan struct{}
}

func NewUsageRecordingService(
    kafkaProducer messaging.KafkaProducer,
    kafkaConsumer messaging.KafkaConsumer,
    subscriptionRepo repositories.SubscriptionRepository,
    usageEventRepo repositories.UsageEventRepository,
    usageProcessingRepo repositories.UsageProcessingStatusRepository,
    logger logger.Logger,
) *UsageRecordingService {
    return &UsageRecordingService{
        kafkaProducer:       kafkaProducer,
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
    
    // 4. Publish to Kafka
    return s.kafkaProducer.PublishUsageEvent(ctx, event)
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
    
    // Insert into TimescaleDB
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

### 8. TimescaleDB Repository Implementation

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
    
    // Get real-time usage summary from continuous aggregates
    GetRealtimeUsage(ctx context.Context, orgID, subscriptionItemID string, 
        since time.Time) (entities.UsageSummary, error)
    
    // Get customer usage summary
    GetCustomerUsage(ctx context.Context, orgID, customerID string, 
        startTime, endTime time.Time) (entities.CustomerUsageSummary, error)
    
    // Get usage analytics by type
    GetUsageTypeAnalytics(ctx context.Context, orgID string, 
        startTime, endTime time.Time) ([]entities.UsageTypeAnalytics, error)
    
    // Refresh continuous aggregates manually (for billing consistency)
    RefreshAggregates(ctx context.Context) error
}
```

#### B. TimescaleDB Repository Implementation

**File**: `internal/infrastructure/db/timescale/usage_event_repository.go`

```go
package timescale

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
    *TimescaleDatabase
    logger logger.Logger
}

func NewUsageEventRepository(usageDb lib.Database, logger logger.Logger) repositories.UsageEventRepository {
    timescaleDB, ok := usageDb.(*TimescaleDatabase)
    if !ok {
        panic("database is not of type *TimescaleDatabase")
    }
    return &UsageEventRepository{
        TimescaleDatabase: timescaleDB,
        logger:            logger,
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

#### C. TimescaleDB Database Connection

**File**: `internal/infrastructure/db/timescale/database.go`

```go
package timescale

import (
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "log"
    "payloop/internal/application/lib/logger"
    "payloop/internal/lib"
)

type TimescaleDatabase struct {
    *pgxpool.Pool
    pgx.Tx
    logger logger.Logger
}

func (r TimescaleDatabase) getTransactionFromContext(ctx context.Context) QueryRower {
    var p QueryRower = r.Pool
    tx := ctx.Value(lib.DBTransaction)
    if tx != nil {
        p = tx.(QueryRower)
    }
    return p
}

func NewTimescaleDatabase(url string, logger logger.Logger) lib.Database {
    logger.Info("Connecting to TimescaleDB", "url", url)

    dbConfig, err := pgxpool.ParseConfig(url)
    if err != nil {
        log.Fatalf("could not parse TimescaleDB config %v", err)
        return nil
    }
    
    pool, err := pgxpool.NewWithConfig(context.TODO(), dbConfig)
    if err != nil {
        log.Fatalf("could not connect to TimescaleDB %v", err)
        return nil
    }

    return &TimescaleDatabase{
        Pool:   pool,
        Tx:     nil,
        logger: logger,
    }
}

func (d *TimescaleDatabase) Ping(ctx context.Context) error {
    return d.Pool.Ping(ctx)
}

func (d *TimescaleDatabase) Close() {
    d.logger.Info("Closing TimescaleDB connection")
    d.Pool.Close()
}

func (d *TimescaleDatabase) Begin(ctx context.Context) (lib.Committer, error) {
    tx, err := d.Pool.Begin(ctx)
    if err != nil {
        return nil, err
    }
    return TimescaleCommitter{
        Tx: tx,
    }, nil
}

type TimescaleCommitter struct {
    pgx.Tx
}

func (c TimescaleCommitter) Commit(ctx context.Context) error {
    return c.Tx.Commit(ctx)
}

func (c TimescaleCommitter) Rollback(ctx context.Context) error {
    return c.Tx.Rollback(ctx)
}

func (c TimescaleCommitter) GetClient() interface{} {
    return c.Tx
}

// QueryRower interface for supporting both Pool and Tx queries
type QueryRower interface {
    Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
    Exec(ctx context.Context, sql string, args ...interface{}) (pgx.CommandTag, error)
    SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}
```

### 9. Environment Configuration

#### A. Environment Variables

Add to `.env`:

```env
# Usage Database
USAGE_DATABASE_URL=postgres://postgres:postgres@localhost:5433/payloop_usage

# Kafka
KAFKA_BROKERS=localhost:9092

# Billing Configuration  
BILLING_GRACE_PERIOD_MINUTES=30
BILLING_SCHEDULE_HOUR=2
```

#### B. Database Scripts

**File**: `scripts/setup-usage-db.sh`

```bash
#!/bin/bash
set -e

echo "Setting up usage database..."

# Generate Prisma client for usage database
echo "Generating Prisma client for usage database..."
pnpm dlx prisma generate --schema=schemas/usage/schema.prisma

# Push schema to usage database
echo "Pushing schema to TimescaleDB..."
pnpm dlx prisma db push --schema=schemas/usage/schema.prisma

echo "Usage database setup completed!"
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
    // Setup test environment with TimescaleDB and Kafka
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
- TimescaleDB query performance
- Billing processing duration
- Usage event lag time
- Continuous aggregate refresh duration

#### B. Health Checks

**File**: `internal/infrastructure/health/usage_health.go`

```go
package health

import (
    "context"
    "database/sql"
    "time"
)

func CheckTimescaleDBHealth(ctx context.Context, db *sql.DB) error {
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
   - Update docker-compose.yml with TimescaleDB and Kafka
   - Create TimescaleDB migrations and continuous aggregates
   - Create usage Prisma schema
   - Remove UsageRecord from main schema

2. **Phase 2: Repository and Domain Layer**
   - Implement TimescaleDB database connection
   - Create usage event repository interfaces
   - Implement TimescaleDB repository implementations
   - Create usage domain entities

3. **Phase 3: Event System and Service Integration** 
   - Implement Kafka producer/consumer infrastructure
   - Create usage event definitions
   - Update usage recording service with Kafka producer
   - Add Kafka consumer goroutine to usage service
   - Create usage aggregation service

4. **Phase 4: Billing Integration**
   - Create simple billing scheduler (no Temporal)
   - Update billing service to use TimescaleDB aggregates
   - Implement 30-minute settlement window

5. **Phase 5: Testing & Deployment**
   - Integration tests with TimescaleDB and Kafka
   - API testing for usage recording
   - Billing accuracy testing
   - Production deployment and monitoring

## Success Criteria

- ✅ Handle current MVP volume (few transactions/day) with room for growth
- ✅ Real-time usage dashboards (< 5 minute latency via continuous aggregates)
- ✅ Accurate billing with 30-minute settlement window
- ✅ Zero data loss during processing (Kafka reliability)
- ✅ Automatic data compression and retention (TimescaleDB policies)
- ✅ Event-driven architecture ready for future scaling
- ✅ Clean separation of usage data from core business operations
- ✅ Simple deployment and monitoring for MVP

## Risks and Mitigations

1. **Event Ordering**: Use partition keys to ensure ordering
2. **Data Loss**: Implement at-least-once delivery with idempotency
3. **Schema Evolution**: Version events and maintain backward compatibility
4. **Cross-Database Consistency**: Use eventual consistency with compensation patterns
5. **Performance**: Monitor and optimize continuous aggregate refresh intervals