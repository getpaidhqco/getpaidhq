---
title: Kafka + NATS Publisher Architecture
description: Comprehensive design document for dual-publisher event system combining Kafka for durable events and NATS for real-time notifications
---

# Kafka + NATS Publisher Architecture

## Executive Summary

This document describes the architecture and implementation of a dual-publisher event system for GetPaidHQ (GPHQ) that combines:

- **Kafka** for durable event storage, audit trails, and downstream processing
- **NATS** for real-time notifications and immediate UI updates

The system maintains clean architecture principles with separate interfaces for different messaging concerns while preserving backward compatibility.

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      Application Services                        │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────┐    ┌─────────────────────────────────┐  │
│  │ NotificationPublisher│    │   DurableEventPublisher       │  │
│  │   (Real-time)       │    │   (Persistent)                │  │
│  └─────────────────────┘    └─────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                    Infrastructure Layer                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────┐    ┌─────────────────────────────────┐  │
│  │      NATS           │    │           Kafka                 │  │
│  │   (In-Memory)       │    │        (Persistent)             │  │
│  │                     │    │                                 │  │
│  │ • Real-time alerts  │    │ • Event sourcing               │  │
│  │ • UI updates        │    │ • Audit trails                 │  │
│  │ • Immediate notify  │    │ • Downstream processing        │  │
│  └─────────────────────┘    └─────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Event Flow Architecture

```
┌─────────────────┐    ┌─────────────────────────────────────────┐
│   API Request   │    │            Service Layer                │
│                 │    │                                         │
│  POST /usage    │───▶│  1. Validate & process business logic  │
│                 │    │  2. Save to database                   │
│                 │    │  3. Create domain event                │
│                 │    │  4. Dual publish:                     │
│                 │    │     • Kafka (durable)                 │
│                 │    │     • NATS (notification)             │
└─────────────────┘    └─────────────────────────────────────────┘
                                          │
                       ┌─────────────────┴─────────────────────┐
                       │                                       │
                       ▼                                       ▼
        ┌─────────────────────────────┐           ┌─────────────────────────────┐
        │         Kafka Topic         │           │        NATS Subject         │
        │    gphq.usage.recorded      │           │      usage.recorded         │
        │                             │           │                             │
        │  • Partitioned by orgId     │           │  • Immediate delivery       │
        │  • Durable storage          │           │  • In-memory               │
        │  • Audit trail              │           │  • Real-time UI updates    │
        │  • Downstream processing    │           │                             │
        └─────────────────────────────┘           └─────────────────────────────┘
                       │                                       │
                       ▼                                       ▼
        ┌─────────────────────────────┐           ┌─────────────────────────────┐
        │    Kafka Consumers          │           │     NATS Subscribers        │
        │                             │           │                             │
        │  • Analytics service        │           │  • Web UI                  │
        │  • Audit service            │           │  • Mobile app              │
        │  • Billing aggregation      │           │  • Real-time dashboards    │
        │  • External webhooks        │           │                             │
        └─────────────────────────────┘           └─────────────────────────────┘
```

## Dual Publishing Strategy Explained

### Why Both Kafka and NATS for the Same Event?

The key architectural decision is using **both publishers for the same business event** because different consumers have fundamentally different requirements:

```
Single Business Event: "Customer Payment Succeeded"
           │
    ┌──────┴──────┐
    │             │
    ▼             ▼
Kafka Event    NATS Event
(Audit Trail)  (Real-time UI)
```

### Different Consumers, Different Needs

#### Example: Payment Success Event

**What happened**: Customer paid $99 for their subscription

**Kafka Consumers (Durable Processing)**:
- **Accounting System**: Complete payment details for tax reporting
- **Analytics Pipeline**: Revenue aggregation, customer LTV analysis
- **Compliance Service**: Audit trail for financial regulations
- **Data Warehouse**: Historical analysis, business intelligence
- **External Webhooks**: Integration with customer's accounting software

**NATS Consumers (Real-time Experience)**:
- **Web Dashboard**: Immediate "Payment Successful" notification
- **Mobile App**: Push notification to customer
- **Customer Portal**: Real-time payment status update
- **Admin Dashboard**: Live payment monitoring
- **Email Service**: Trigger immediate receipt email

### Concrete Implementation Example

```go
// Business Logic: Process Payment
func (s *PaymentService) ProcessPayment(ctx context.Context, paymentId string) error {
    // 1. Execute business logic
    payment, err := s.chargeCustomer(ctx, paymentId)
    if err != nil {
        return err
    }

    // 2. Persist state change
    if err := s.repository.UpdatePayment(ctx, payment); err != nil {
        return err
    }

    // 3. Kafka Event: Complete audit record
    kafkaEvent := events.PaymentEvent{
        BaseEvent: events.BaseEvent{
            EventId:       "evt_123",
            EventType:     "payment.succeeded",
            OrgId:         payment.OrgId,
            AggregateId:   payment.Id,
            Timestamp:     time.Now(),
        },
        PaymentId:         payment.Id,
        CustomerId:        payment.CustomerId,
        Amount:            payment.Amount,
        Currency:          payment.Currency,
        PaymentMethod:     payment.PaymentMethod,
        ProcessorResponse: payment.ProcessorResponse, // Full audit data
        BillingAddress:    payment.BillingAddress,
        TaxAmount:         payment.TaxAmount,
        NetAmount:         payment.NetAmount,
        // Complete business context for compliance
    }
    s.durablePublisher.PublishPaymentEvent(ctx, kafkaEvent)

    // 4. NATS Event: Minimal real-time notification
    natsPayload := map[string]interface{}{
        "payment_id":     payment.Id,
        "customer_id":    payment.CustomerId,
        "amount":         payment.Amount,
        "status":         "succeeded",
        "message":        "Payment processed successfully",
        "display_amount": "$99.00",
        // Just what the UI needs immediately
    }
    s.notificationPublisher.Publish(payment.OrgId, "payment.success", natsPayload)

    return nil
}
```

### Publisher Responsibilities

#### Kafka Events: "What Happened for the Record"

**Purpose**: Permanent, immutable record of business events
**Data Format**: Complete, structured events with full business context
**Consumers**: Batch processors, analytics systems, compliance auditors

```go
// Kafka: Full business context and audit trail
type PaymentEvent struct {
    BaseEvent
    PaymentId          string             `json:"payment_id"`
    SubscriptionId     string             `json:"subscription_id,omitempty"`
    CustomerId         string             `json:"customer_id"`
    InvoiceId          string             `json:"invoice_id,omitempty"`
    Amount             int64              `json:"amount"`
    Currency           string             `json:"currency"`
    PaymentMethod      string             `json:"payment_method"`
    ProcessorResponse  map[string]string  `json:"processor_response"`
    BillingAddress     Address            `json:"billing_address"`
    TaxAmount          int64              `json:"tax_amount"`
    FeesAmount         int64              `json:"fees_amount"`
    NetAmount          int64              `json:"net_amount"`
    ProcessedAt        time.Time          `json:"processed_at"`
    // Everything needed for audit, compliance, analytics
}

// Kafka Event Uses:
// • Accounting: Calculate taxes, fees, net revenue
// • Analytics: Customer LTV, revenue trends, churn analysis  
// • Compliance: Audit logs, regulatory reporting
// • Data Warehouse: Historical analysis, business intelligence
// • Billing: Invoice reconciliation, revenue recognition
```

#### NATS Events: "Tell Users What Happened Now"

**Purpose**: Immediate feedback to users and real-time systems
**Data Format**: Minimal, user-focused information for immediate consumption
**Consumers**: Real-time UIs, notifications, immediate user feedback

```go
// NATS: User-focused, minimal payload for immediate consumption
natsPayload := map[string]interface{}{
    "payment_id":     "pay_123",
    "customer_id":    "cus_456", 
    "amount":         9900,        // cents
    "currency":       "USD",
    "status":         "succeeded",
    "message":        "Payment processed successfully",
    "display_amount": "$99.00",    // formatted for display
    "timestamp":      time.Now(),
    // Just what the UI needs for immediate feedback
}

// NATS Event Uses:
// • Web Dashboard: Show success toast notification
// • Mobile App: Push notification to customer
// • Customer Portal: Update payment status badge in real-time
// • Email Service: Trigger immediate receipt email
// • Support Dashboard: Live customer payment status updates
```

### Real-World Event Examples

#### Usage Recording Event

**Business Event**: Customer made 1,000 API calls

**Kafka Event** (`gphq.usage.recorded`):
```json
{
  "event_id": "evt_usage_123",
  "event_type": "usage.recorded", 
  "org_id": "org_stripe",
  "aggregate_id": "ur_789",
  "timestamp": "2024-01-15T10:30:00Z",
  "subscription_id": "sub_pro_plan",
  "subscription_item_id": "si_api_calls",
  "customer_id": "cus_456",
  "usage_record": {
    "id": "ur_789",
    "quantity": 1000,
    "timestamp": "2024-01-15T10:30:00Z",
    "metadata": {"endpoint": "charges", "version": "v1", "ip": "192.168.1.1"}
  },
  "metric_name": "api_calls",
  "unit_price": 1,
  "billing_period": "2024-01"
}
```
**Consumers**: Billing aggregation, usage analytics, compliance audit, invoice generation

**NATS Event** (`usage.recorded`):
```json
{
  "id": "evt_456",
  "org_id": "org_stripe", 
  "topic": "usage.recorded",
  "data": {
    "subscription_id": "sub_pro_plan",
    "quantity": 1000,
    "metric_display": "API Calls",
    "current_total": 15000,
    "monthly_limit": 100000,
    "percentage_used": 15,
    "display_message": "1,000 API calls recorded"
  },
  "created_at": "2024-01-15T10:30:00Z"
}
```
**Consumers**: Usage dashboard updates, real-time progress bars, threshold alert system

#### Subscription Cancellation Event

**Business Event**: Customer cancelled their Pro subscription

**Kafka Event** (`gphq.billing.events`):
```json
{
  "event_type": "subscription.cancelled",
  "event_id": "evt_sub_cancel_123",
  "org_id": "org_acme",
  "aggregate_id": "sub_123",
  "timestamp": "2024-01-15T14:22:00Z", 
  "subscription_id": "sub_123",
  "customer_id": "cus_456",
  "subscription_event_type": "cancelled",
  "previous_status": "active",
  "new_status": "cancelled",
  "change_reason": "customer_request",
  "effective_date": "2024-02-01T00:00:00Z",
  "cancellation_details": {
    "cancelled_by": "customer",
    "cancellation_reason": "cost_too_high",
    "feedback": "Found a cheaper alternative",
    "retention_attempted": true,
    "discount_offered": 2000
  },
  "subscription": {
    "id": "sub_123",
    "plan": "pro_monthly",
    "billing_cycle": "monthly", 
    "current_period_start": "2024-01-01T00:00:00Z",
    "current_period_end": "2024-02-01T00:00:00Z",
    "items": [...],
    "total_revenue": 29700
  },
  "proration_amount": 0,
  "refund_amount": 0
}
```
**Consumers**: Churn analysis, billing proration calculations, revenue recognition, customer lifecycle analytics

**NATS Event** (`subscription.cancelled`):
```json
{
  "id": "evt_789",
  "org_id": "org_acme",
  "topic": "subscription.cancelled", 
  "data": {
    "subscription_id": "sub_123",
    "customer_id": "cus_456",
    "status": "cancelled",
    "effective_date": "2024-02-01",
    "plan_name": "Pro Monthly",
    "message": "Subscription cancelled successfully",
    "access_until": "2024-02-01T00:00:00Z"
  },
  "created_at": "2024-01-15T14:22:00Z"
}
```
**Consumers**: Customer dashboard cancellation confirmation, admin alert notifications, support team updates

### Publisher Selection Decision Tree

```
Business Event Occurs
        │
        ▼
┌─────────────────┐
│ Always publish  │ 
│ to Kafka for    │──────┐
│ audit trail     │      │
└─────────────────┘      │
        │                │
        ▼                ▼
┌─────────────────┐  ┌─────────────────┐
│ Does it need    │  │ Kafka Event:    │
│ real-time user  │  │ • Complete data │
│ feedback?       │  │ • Audit trail   │
└─────────────────┘  │ • Compliance    │
        │            │ • Analytics     │
   ┌────┴────┐       └─────────────────┘
   │ Yes│ No │              
   ▼    │    │              
┌─────────────────┐         │
│ Also publish    │         │
│ to NATS for     │         │
│ immediate UI    │         │
│ updates         │         │
└─────────────────┘         │
        │                   │
        ▼                   │
┌─────────────────┐         │
│ NATS Event:     │         │
│ • Minimal data  │         │
│ • User-focused  │         │
│ • Immediate     │         │
│ • UI updates    │         │
└─────────────────┘         │
        │                   │
        └───────────────────┘
                │
                ▼
        ┌─────────────────┐
        │ Event processing│
        │ complete        │
        └─────────────────┘
```

### Why Not Single Publisher?

#### Option 1: Kafka Only ❌
```go
// Problems with Kafka-only approach:
// 1. UI responsiveness: Users wait for Kafka roundtrip (200-500ms)
// 2. Complexity: Real-time consumers must handle heavy audit data
// 3. Coupling: UI logic mixed with compliance/audit requirements
// 4. Performance: Real-time systems slowed by analytical data
// 5. Cost: All consumers process full event payload regardless of need
```

#### Option 2: NATS Only ❌
```go
// Problems with NATS-only approach:
// 1. Data loss: No audit trail when messages are consumed
// 2. Compliance risk: No permanent record for regulatory requirements
// 3. Analytics loss: Historical data unavailable if systems are down
// 4. Event sourcing impossible: Cannot replay business events
// 5. Disaster recovery: No durable event log for system reconstruction
```

#### Option 3: Dual Publishers ✅
```go
// Benefits of dual approach:
// 1. Performance: Fast UI responses via NATS (10-50ms)
// 2. Reliability: Permanent audit trail via Kafka
// 3. Separation: Each consumer gets exactly what it needs
// 4. Compliance: Full audit trail always maintained
// 5. Scalability: Systems can be optimized for their specific use case
// 6. Flexibility: Easy to add new consumer types without affecting others
```

## Component Architecture

### 1. Application Layer Interfaces

#### NotificationPublisher Interface
```go
// Purpose: Real-time notifications and immediate updates
type NotificationPublisher interface {
    Publish(orgId string, topic string, message interface{}) error
    Subscribe(topic string, handler func(topic string, data []byte)) (Subscription, error)
}

// Payload Structure (flexible)
type NotificationPayload struct {
    Id        string      `json:"id"`
    OrgId     string      `json:"org_id"`
    Topic     string      `json:"topic"`
    Data      interface{} `json:"data"`
    CreatedAt time.Time   `json:"created_at"`
}
```

#### DurableEventPublisher Interface
```go
// Purpose: Event sourcing, audit trails, and downstream processing
type DurableEventPublisher interface {
    PublishUsageEvent(ctx context.Context, event UsageRecordedEvent) error
    PublishBillingEvent(ctx context.Context, event BillingEvent) error
    PublishPaymentEvent(ctx context.Context, event PaymentEvent) error
}
```

### 2. Domain Events Structure

#### Base Event Schema
```go
type BaseEvent struct {
    EventId          string            `json:"event_id"`          // Unique event identifier
    EventType        string            `json:"event_type"`        // e.g., "usage.recorded"
    OrgId            string            `json:"org_id"`            // Tenant identifier
    AggregateId      string            `json:"aggregate_id"`      // Entity ID
    AggregateType    string            `json:"aggregate_type"`    // Entity type
    AggregateVersion int               `json:"aggregate_version"` // Event version
    Timestamp        time.Time         `json:"timestamp"`         // Event timestamp
    Metadata         map[string]string `json:"metadata,omitempty"` // Additional context
}
```

#### Specific Event Types
```go
// Usage recording events
type UsageRecordedEvent struct {
    BaseEvent
    SubscriptionId     string                `json:"subscription_id"`
    SubscriptionItemId string                `json:"subscription_item_id"`
    CustomerId         string                `json:"customer_id"`
    UsageRecord        entities.UsageRecord  `json:"usage_record"`
}

// Billing events
type BillingEvent struct {
    BaseEvent
    SubscriptionId string `json:"subscription_id"`
    InvoiceId      string `json:"invoice_id,omitempty"`
    Amount         int64  `json:"amount"`
    Currency       string `json:"currency"`
}

// Payment events
type PaymentEvent struct {
    BaseEvent
    PaymentId      string             `json:"payment_id"`
    SubscriptionId string             `json:"subscription_id,omitempty"`
    Payment        entities.Payment   `json:"payment"`
}
```

### 3. Infrastructure Layer

#### NATS Implementation
```go
type NatsNotificationPublisher struct {
    *nats.Conn
    logger logger.Logger
}

// Features:
// • Embedded server for development
// • Subject-based routing
// • In-memory message delivery
// • Low latency for real-time updates
```

#### Kafka Implementation
```go
type kafkaEventPublisher struct {
    producer sarama.SyncProducer
    logger   logger.Logger
    config   Config
}

// Features:
// • Persistent message storage
// • Partitioned by orgId for tenant isolation
// • Configurable retention policies
// • High throughput for event processing
```

## Topic Design Strategy

### Kafka Topics

#### Topic Naming Convention
- Format: `gphq.{domain}.{event}`
- Examples:
  - `gphq.usage.recorded`
  - `gphq.billing.events`
  - `gphq.payment.events`

#### Partitioning Strategy
- **Partition Key**: `orgId` (tenant identifier)
- **Benefits**:
  - Tenant isolation within topics
  - Consistent ordering per tenant
  - Scalable consumer groups
  - Efficient resource utilization

#### Topic Configuration
```yaml
# Production Configuration
topics:
  gphq.usage.recorded:
    partitions: 12
    replication_factor: 3
    retention_ms: 604800000  # 7 days
    cleanup_policy: delete
    
  gphq.billing.events:
    partitions: 12
    replication_factor: 3
    retention_ms: 2592000000  # 30 days
    cleanup_policy: delete
    
  gphq.payment.events:
    partitions: 12
    replication_factor: 3
    retention_ms: 2592000000  # 30 days
    cleanup_policy: delete
```

### NATS Subjects

#### Subject Naming
- Format: `{domain}.{event}` (existing pattern)
- Examples:
  - `usage.recorded`
  - `billing.invoice.created`
  - `payment.processed`

#### Subject Hierarchy
```
usage.*
├── usage.recorded
├── usage.aggregated
└── usage.reset

billing.*
├── billing.invoice.created
├── billing.invoice.paid
└── billing.invoice.failed

payment.*
├── payment.created
├── payment.processed
└── payment.failed
```

## Service Integration Patterns

### Dual Publishing Pattern
```go
func (s *UsageRecordingService) RecordUsage(ctx context.Context, orgId string, input dto.RecordUsageInput) (entities.UsageRecord, error) {
    // 1. Business logic and persistence
    usageRecord, err := s.createAndSaveUsageRecord(ctx, orgId, input)
    if err != nil {
        return entities.UsageRecord{}, err
    }

    // 2. Publish durable event (Kafka) - async, don't fail operation
    durableEvent := events.NewUsageRecordedEvent(orgId, usageRecord)
    if err := s.durablePublisher.PublishUsageEvent(ctx, durableEvent); err != nil {
        s.logger.Warn("Failed to publish durable usage event", "error", err)
        // Continue - don't fail the operation
    }

    // 3. Publish notification (NATS) - async, don't fail operation
    if err := s.notificationPublisher.Publish(orgId, "usage.recorded", usageRecord); err != nil {
        s.logger.Warn("Failed to publish usage notification", "error", err)
        // Continue - don't fail the operation
    }

    return usageRecord, nil
}
```

### Error Handling Strategy
```go
type PublishingStrategy int

const (
    BestEffort PublishingStrategy = iota  // Log errors, continue
    Guaranteed                           // Fail operation on publish error
    Deferred                            // Queue for later retry
)

func (s *BaseService) publishEvents(ctx context.Context, strategy PublishingStrategy, events ...Event) error {
    var errs []error
    
    for _, event := range events {
        if err := s.publish(ctx, event); err != nil {
            errs = append(errs, err)
            
            switch strategy {
            case BestEffort:
                s.logger.Warn("Event publish failed", "error", err)
                continue
            case Guaranteed:
                return fmt.Errorf("critical event publish failed: %w", err)
            case Deferred:
                s.queueForRetry(event)
                continue
            }
        }
    }
    
    return nil
}
```

## Data Flow Diagrams

### Usage Recording Flow
```
┌─────────────────┐    ┌─────────────────────────────────────────┐
│   Web Client    │    │             API Gateway                 │
│                 │    │                                         │
│  POST /usage    │───▶│  Authentication & Authorization         │
│  {              │    │  Rate limiting                          │
│    "quantity":  │    │  Request validation                     │
│    "100",       │    │                                         │
│    "timestamp": │    │                                         │
│    "2024-01-15" │    │                                         │
│  }              │    │                                         │
└─────────────────┘    └─────────────────────────────────────────┘
                                          │
                                          ▼
                       ┌─────────────────────────────────────────┐
                       │       Usage Recording Service           │
                       │                                         │
                       │  1. Validate input                     │
                       │  2. Create UsageRecord entity          │
                       │  3. Save to database                   │
                       │  4. Create domain event                │
                       │  5. Publish to Kafka                   │
                       │  6. Publish to NATS                    │
                       │  7. Return response                    │
                       └─────────────────────────────────────────┘
                                          │
                       ┌─────────────────┴─────────────────────┐
                       │                                       │
                       ▼                                       ▼
        ┌─────────────────────────────┐           ┌─────────────────────────────┐
        │         Kafka               │           │          NATS               │
        │   gphq.usage.recorded       │           │     usage.recorded          │
        │                             │           │                             │
        │  Event: {                   │           │  Payload: {                 │
        │    "event_id": "evt_123",   │           │    "id": "evt_456",         │
        │    "event_type": "usage.    │           │    "org_id": "org_abc",     │
        │     recorded",              │           │    "topic": "usage.         │
        │    "org_id": "org_abc",     │           │     recorded",              │
        │    "aggregate_id": "ur_789",│           │    "data": {...},           │
        │    "timestamp": "2024-01-15"│           │    "created_at": "2024-01-15"│
        │    "usage_record": {...}    │           │  }                          │
        │  }                          │           │                             │
        └─────────────────────────────┘           └─────────────────────────────┘
                       │                                       │
                       ▼                                       ▼
        ┌─────────────────────────────┐           ┌─────────────────────────────┐
        │    Downstream Systems       │           │      Real-time Systems      │
        │                             │           │                             │
        │  • Billing aggregation      │           │  • Web dashboard            │
        │  • Analytics pipeline       │           │  • Mobile notifications     │
        │  • Audit logging            │           │  • Live usage meters        │
        │  • External webhooks        │           │                             │
        └─────────────────────────────┘           └─────────────────────────────┘
```

### Billing Event Flow
```
┌─────────────────┐    ┌─────────────────────────────────────────┐
│   Subscription  │    │           Billing Service               │
│   Workflow      │    │                                         │
│                 │───▶│  1. Calculate billing amount           │
│  "Charge        │    │  2. Create invoice                     │
│   subscription" │    │  3. Process payment                    │
│                 │    │  4. Update subscription status         │
│                 │    │  5. Emit billing events               │
└─────────────────┘    └─────────────────────────────────────────┘
                                          │
                                          ▼
                       ┌─────────────────────────────────────────┐
                       │          Event Publishing               │
                       │                                         │
                       │  Kafka: BillingEvent                   │
                       │  {                                     │
                       │    "event_type": "billing.charged",   │
                       │    "subscription_id": "sub_123",      │
                       │    "invoice_id": "inv_456",           │
                       │    "amount": 2999,                    │
                       │    "currency": "USD"                  │
                       │  }                                     │
                       │                                         │
                       │  NATS: billing.charged                 │
                       │  {                                     │
                       │    "subscription_id": "sub_123",      │
                       │    "amount": 2999,                    │
                       │    "status": "charged"                │
                       │  }                                     │
                       └─────────────────────────────────────────┘
```

## Infrastructure Requirements

### Kafka Cluster
```yaml
# Development Environment
kafka:
  brokers: ["localhost:9092"]
  replication_factor: 1
  partitions: 3
  retention_ms: 86400000  # 1 day

# Production Environment
kafka:
  brokers: 
    - "kafka-1.internal:9092"
    - "kafka-2.internal:9092"
    - "kafka-3.internal:9092"
  replication_factor: 3
  partitions: 12
  retention_ms: 604800000  # 7 days
  
  # Performance tuning
  batch_size: 16384
  linger_ms: 10
  compression_type: "gzip"
  acks: "all"
  retries: 2147483647
```

### NATS Server
```yaml
# Embedded for development
nats:
  embedded: true
  port: 4222
  
# Production cluster
nats:
  servers:
    - "nats-1.internal:4222"
    - "nats-2.internal:4222"
    - "nats-3.internal:4222"
  cluster: "gphq-cluster"
  max_payload: 1048576  # 1MB
  max_connections: 64000
```

## Monitoring and Observability

### Metrics Collection
```go
type PublisherMetrics struct {
    KafkaPublishTotal     prometheus.Counter
    KafkaPublishErrors    prometheus.Counter
    KafkaPublishDuration  prometheus.Histogram
    NatsPublishTotal      prometheus.Counter
    NatsPublishErrors     prometheus.Counter
    NatsPublishDuration   prometheus.Histogram
}

// Example metrics
gphq_kafka_events_published_total{topic="gphq.usage.recorded", org_id="org_123"}
gphq_kafka_publish_duration_seconds{topic="gphq.usage.recorded"}
gphq_nats_messages_published_total{subject="usage.recorded"}
```

### Health Checks
```go
type HealthChecker struct {
    kafkaPublisher KafkaPublisher
    natsPublisher  NatsPublisher
}

func (h *HealthChecker) CheckHealth() HealthStatus {
    return HealthStatus{
        Kafka: h.checkKafkaHealth(),
        NATS:  h.checkNatsHealth(),
    }
}

// Health check endpoints
GET /health/kafka   → Kafka broker connectivity
GET /health/nats    → NATS server connectivity
GET /health/events  → Overall event system health
```

### Logging Strategy
```go
// Structured logging for events
logger.Info("Event published successfully",
    "event_type", event.EventType,
    "org_id", event.OrgId,
    "aggregate_id", event.AggregateId,
    "publisher", "kafka",
    "topic", topicName,
    "partition", partition,
    "offset", offset,
)

// Error logging with context
logger.Error("Failed to publish event",
    "error", err,
    "event_type", event.EventType,
    "org_id", event.OrgId,
    "publisher", "kafka",
    "retry_count", retryCount,
)
```

## Security Considerations

### Authentication & Authorization
```yaml
# Kafka SASL configuration
kafka:
  security:
    protocol: "SASL_SSL"
    sasl_mechanism: "PLAIN"
    username: "gphq-service"
    password: "${KAFKA_PASSWORD}"
    
# NATS authentication
nats:
  auth:
    token: "${NATS_TOKEN}"
    # or JWT-based authentication
    jwt: "${NATS_JWT}"
```

### Data Encryption
```go
// Event payload encryption (if needed)
type EncryptedEvent struct {
    BaseEvent
    EncryptedPayload string `json:"encrypted_payload"`
    EncryptionAlgorithm string `json:"encryption_algorithm"`
}

// PII handling
func (e *UsageRecordedEvent) Sanitize() {
    // Remove or hash sensitive data
    e.UsageRecord.CustomerEmail = ""
    e.UsageRecord.CustomerName = hash(e.UsageRecord.CustomerName)
}
```

### Access Control
```go
// Topic-based access control
func (k *KafkaPublisher) authorize(orgId, topic string) error {
    if !k.acl.CanPublish(orgId, topic) {
        return ErrUnauthorized
    }
    return nil
}

// Tenant isolation in consumers
func (c *Consumer) processMessage(msg *sarama.ConsumerMessage) error {
    var event BaseEvent
    if err := json.Unmarshal(msg.Value, &event); err != nil {
        return err
    }
    
    if !c.isAuthorizedForOrg(event.OrgId) {
        return ErrUnauthorized
    }
    
    return c.handleEvent(event)
}
```

## Performance Considerations

### Throughput Optimization
```go
// Batch publishing for high throughput
func (k *KafkaPublisher) PublishBatch(ctx context.Context, events []Event) error {
    messages := make([]*sarama.ProducerMessage, 0, len(events))
    
    for _, event := range events {
        data, _ := json.Marshal(event)
        messages = append(messages, &sarama.ProducerMessage{
            Topic: k.getTopicName(event.EventType),
            Key:   sarama.StringEncoder(event.OrgId),
            Value: sarama.ByteEncoder(data),
        })
    }
    
    return k.producer.SendMessages(messages)
}
```

### Memory Management
```go
// Connection pooling
type ConnectionPool struct {
    kafkaProducers chan sarama.SyncProducer
    natsConns      chan *nats.Conn
}

// Resource cleanup
func (p *Publisher) Close() error {
    var errs []error
    
    if err := p.kafkaProducer.Close(); err != nil {
        errs = append(errs, err)
    }
    
    if err := p.natsConn.Close(); err != nil {
        errs = append(errs, err)
    }
    
    return errors.Join(errs...)
}
```

## Migration Strategy

### Phase 1: Foundation (Week 1)
- [ ] Rename PubSub → NotificationPublisher
- [ ] Create domain events structure  
- [ ] Update existing NATS implementation
- [ ] Add DurableEventPublisher interface

### Phase 2: Kafka Implementation (Week 2)
- [ ] Implement Kafka publisher
- [ ] Configure topics and partitions
- [ ] Add health checks and monitoring
- [ ] Create comprehensive tests

### Phase 3: Service Integration (Week 3)
- [ ] Update core services to use dual publishing
- [ ] Implement error handling strategies
- [ ] Add performance monitoring
- [ ] Documentation and examples

### Phase 4: Production Readiness (Week 4)
- [ ] Load testing and optimization
- [ ] Security hardening
- [ ] Operational runbooks
- [ ] Team training

## Conclusion

The Kafka + NATS dual-publisher architecture provides:

1. **Separation of Concerns**: Different systems for different purposes
2. **Reliability**: Durable event storage with real-time notifications
3. **Scalability**: Partitioned topics with efficient resource utilization
4. **Maintainability**: Clean interfaces and dependency injection
5. **Observability**: Comprehensive monitoring and logging
6. **Security**: Proper authentication, authorization, and data protection

This architecture enables GPHQ to build robust, scalable event-driven systems while maintaining clean architecture principles and operational excellence.