# Kafka + NATS Publisher Implementation Specification

## Overview

This specification defines the implementation of a dual-publisher event system combining Kafka for durable event storage and NATS for real-time notifications. The system maintains clean architecture principles with separate interfaces for different messaging concerns.

## Architecture Goals

1. **Separation of Concerns**: Different publishers for different purposes
2. **Clean Architecture**: Interfaces in application layer, implementations in infrastructure
3. **DDD Compliance**: Domain events with proper structure and validation
4. **Backward Compatibility**: No breaking changes to existing NATS implementation
5. **Multi-tenancy**: Org-scoped events with proper isolation

## Current State Analysis

### Existing Implementation
- **PubSub Interface**: Single interface for publish/subscribe operations
- **NATS Implementation**: Real-time notifications with embedded server
- **Event Topics**: 100+ predefined topics in `internal/application/lib/events/topic/`
- **Payload Structure**: Standardized with `orgId`, `topic`, `data`, `timestamp`

### Limitations
- No durable event storage for audit trails
- Limited event structure for complex domain events
- Single messaging system (NATS only)
- Interface naming confusion ("PubSub" vs actual purpose)

## Design Decisions

### 1. Dual Publishing Strategy Rationale

The core architectural decision is to use **both publishers for the same business event** because different consumers have fundamentally different requirements:

#### Why Both Kafka and NATS?

**The Problem**: A single business event (e.g., "Payment Succeeded") serves multiple purposes:
- **Immediate user feedback** (real-time dashboard updates, notifications)
- **Audit compliance** (financial records, regulatory reporting)
- **Business analytics** (revenue trends, customer behavior)
- **System integration** (external webhooks, downstream services)

**The Solution**: Dual publishing addresses these diverse needs efficiently:

```
Business Event: "Customer Payment Succeeded"
           │
    ┌──────┴──────┐
    │             │
    ▼             ▼
Kafka Event    NATS Event
(Audit Trail)  (Real-time UI)
```

#### Publisher Responsibilities

**NotificationPublisher** (NATS) - "Tell Users Now"
- Purpose: Real-time notifications, UI updates, immediate alerts
- Implementation: NATS
- Payload: Minimal, user-focused data
- Durability: In-memory, temporary
- Latency: 10-50ms
- Consumers: Web dashboards, mobile apps, real-time notifications

**DurableEventPublisher** (Kafka) - "Remember What Happened"
- Purpose: Event sourcing, audit trails, downstream processing
- Implementation: Kafka
- Payload: Complete, structured domain events
- Durability: Persistent, partitioned, replicated
- Latency: 100-500ms (acceptable for batch processing)
- Consumers: Analytics, compliance, billing, external integrations

#### Concrete Example: Payment Processing

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
            EventId:     "evt_123",
            EventType:   "payment.succeeded",
            OrgId:       payment.OrgId,
            Timestamp:   time.Now(),
        },
        PaymentId:         payment.Id,
        Amount:            payment.Amount,
        Currency:          payment.Currency,
        ProcessorResponse: payment.ProcessorResponse, // Full audit data
        BillingAddress:    payment.BillingAddress,
        TaxAmount:         payment.TaxAmount,
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

#### Benefits of Dual Publishing

1. **Performance Separation**: 
   - UI gets fast responses (NATS: 10-50ms)
   - Analytics gets complete data (Kafka: durable, structured)

2. **Concern Separation**:
   - Real-time systems handle minimal payloads
   - Audit systems handle complete business context
   - No coupling between UI and compliance requirements

3. **Reliability**:
   - NATS failure doesn't affect audit trail
   - Kafka failure doesn't affect user experience
   - Each system optimized for its specific use case

4. **Compliance**:
   - Kafka ensures permanent audit trail
   - Structured events support regulatory requirements
   - Complete business context preserved

#### Publisher Selection Rules

**Use Kafka When**:
- Event needs permanent audit trail
- Compliance/regulatory requirements
- Complete business context required
- Event sourcing/replay capability needed
- Analytics and business intelligence

**Use NATS When**:
- Immediate user feedback required
- Real-time dashboard updates
- Temporary status notifications
- UI progress indicators
- System health alerts

**Use Both When**:
- Critical business events (payments, subscriptions)
- Customer-facing actions (need both audit + feedback)
- State changes affecting multiple systems
- Events requiring both compliance and user experience

### 2. Topic Strategy

**Standard Topics with OrgId in Payload**
- No tenant-based topic prefixes
- Topic format: `gphq.{domain}.{event}` (e.g., `gphq.usage.recorded`)
- Tenant isolation via partitioning by `orgId`
- Benefits: Scalable, analytics-friendly, manageable

### 3. Event Structure

**Domain Events** (New)
- Structured schema with `BaseEvent` + specific event data
- Event sourcing compatible
- Versioning support
- Metadata for extensibility

## Implementation Plan

### Phase 1: Interface Refactoring
1. Rename `PubSub` → `NotificationPublisher`
2. Update all references in infrastructure layer
3. Maintain backward compatibility

### Phase 2: Domain Events
1. Create `BaseEvent` structure
2. Define specific event types for all business domains
3. Implement event factory functions
4. Map events to appropriate publishers (Kafka vs NATS)

### Phase 3: Kafka Publisher
1. Create `DurableEventPublisher` interface
2. Implement Kafka publisher with Sarama
3. Configure topic management and partitioning

### Phase 4: Service Integration
1. Update services to use both publishers
2. Implement dual-publishing pattern
3. Add error handling and fallback

### Phase 5: Testing & Validation
1. Unit tests for new implementations
2. Integration tests for dual-publishing
3. Performance testing

## File Structure

```
internal/application/lib/events/
├── notification_publisher.go  # Renamed from pubsub.go
├── durable_publisher.go       # New - Kafka interface
├── domain_events.go           # New - Structured events
└── topic/
    └── topics.go              # Existing - no changes

internal/infrastructure/events/
├── kafka/
│   ├── fx.go                  # New - DI config
│   ├── kafka_publisher.go     # New - Kafka implementation
│   └── config.go              # New - Kafka config
└── nats/
    ├── fx.go                  # Update - interface name
    ├── nats.go                # Update - interface name
    └── nats_test.go           # Update - interface name
```

## Interface Definitions

### NotificationPublisher
```go
type NotificationPublisher interface {
    Publish(orgId string, topic string, message interface{}) error
    Subscribe(topic string, handler func(topic string, data []byte)) (Subscription, error)
}
```

### DurableEventPublisher
```go
type DurableEventPublisher interface {
    // Core Business Events - Audit Trail & Compliance
    PublishUsageEvent(ctx context.Context, event UsageRecordedEvent) error
    PublishBillingEvent(ctx context.Context, event BillingEvent) error
    PublishPaymentEvent(ctx context.Context, event PaymentEvent) error
    PublishSubscriptionEvent(ctx context.Context, event SubscriptionEvent) error
    PublishCustomerEvent(ctx context.Context, event CustomerEvent) error
    PublishInvoiceEvent(ctx context.Context, event InvoiceEvent) error
    PublishRefundEvent(ctx context.Context, event RefundEvent) error
    
    // Batch operations for high-volume events
    PublishUsageBatch(ctx context.Context, events []UsageRecordedEvent) error
    PublishEventBatch(ctx context.Context, events []BaseEvent) error
}
```

### BaseEvent
```go
type BaseEvent struct {
    EventId          string            `json:"event_id"`
    EventType        string            `json:"event_type"`
    OrgId            string            `json:"org_id"`
    AggregateId      string            `json:"aggregate_id"`
    AggregateType    string            `json:"aggregate_type"`
    AggregateVersion int               `json:"aggregate_version"`
    Timestamp        time.Time         `json:"timestamp"`
    Metadata         map[string]string `json:"metadata,omitempty"`
}
```

## Complete Event Specification

### Event Publisher Assignment Strategy

**Kafka Events (Durable, Audit Trail)**
- **Purpose**: Compliance, audit trails, event sourcing, downstream analytics
- **Retention**: 7-30 days depending on compliance requirements
- **Characteristics**: Structured, versioned, immutable

**NATS Events (Real-time Notifications)**
- **Purpose**: UI updates, real-time dashboards, immediate notifications
- **Retention**: In-memory only
- **Characteristics**: Flexible payload, immediate delivery

### Domain Event Definitions

#### 1. Usage & Metering Events

**Kafka Events:**
```go
type UsageRecordedEvent struct {
    BaseEvent
    SubscriptionId     string                `json:"subscription_id"`
    SubscriptionItemId string                `json:"subscription_item_id"`
    CustomerId         string                `json:"customer_id"`
    UsageRecord        entities.UsageRecord  `json:"usage_record"`
    MetricName         string                `json:"metric_name"`
    Quantity           int64                 `json:"quantity"`
    UnitPrice          int64                 `json:"unit_price,omitempty"`
    BillingPeriod      string                `json:"billing_period"`
}

type UsageBatchRecordedEvent struct {
    BaseEvent
    BatchId            string                   `json:"batch_id"`
    Records            []entities.UsageRecord   `json:"records"`
    BatchSize          int                      `json:"batch_size"`
    TotalQuantity      int64                    `json:"total_quantity"`
    ProcessingTimeMs   int64                    `json:"processing_time_ms"`
}

type UsageAggregatedEvent struct {
    BaseEvent
    SubscriptionId     string    `json:"subscription_id"`
    MetricName         string    `json:"metric_name"`
    TotalUsage         int64     `json:"total_usage"`
    BillingPeriodStart time.Time `json:"billing_period_start"`
    BillingPeriodEnd   time.Time `json:"billing_period_end"`
}
```

**NATS Events:**
- `usage.recorded` - Real-time usage updates for dashboards
- `usage.threshold.warning` - Usage approaching limits
- `usage.threshold.exceeded` - Usage limits exceeded

#### 2. Billing & Subscription Events

**Kafka Events:**
```go
type BillingEvent struct {
    BaseEvent
    BillingEventType   string    `json:"billing_event_type"` // "invoice_created", "payment_charged", etc.
    SubscriptionId     string    `json:"subscription_id"`
    CustomerId         string    `json:"customer_id"`
    InvoiceId          string    `json:"invoice_id,omitempty"`
    Amount             int64     `json:"amount"`
    Currency           string    `json:"currency"`
    BillingPeriodStart time.Time `json:"billing_period_start"`
    BillingPeriodEnd   time.Time `json:"billing_period_end"`
    TaxAmount          int64     `json:"tax_amount,omitempty"`
    DiscountAmount     int64     `json:"discount_amount,omitempty"`
}

type SubscriptionEvent struct {
    BaseEvent
    SubscriptionEventType string                    `json:"subscription_event_type"`
    SubscriptionId        string                    `json:"subscription_id"`
    CustomerId           string                    `json:"customer_id"`
    PreviousStatus       string                    `json:"previous_status,omitempty"`
    NewStatus            string                    `json:"new_status"`
    Subscription         entities.Subscription     `json:"subscription"`
    ChangeReason         string                    `json:"change_reason,omitempty"`
    EffectiveDate        time.Time                 `json:"effective_date"`
}

type InvoiceEvent struct {
    BaseEvent
    InvoiceEventType   string            `json:"invoice_event_type"`
    InvoiceId          string            `json:"invoice_id"`
    SubscriptionId     string            `json:"subscription_id"`
    CustomerId         string            `json:"customer_id"`
    Invoice            entities.Invoice  `json:"invoice"`
    Amount             int64             `json:"amount"`
    Currency           string            `json:"currency"`
    DueDate            time.Time         `json:"due_date"`
    PaidDate           *time.Time        `json:"paid_date,omitempty"`
}
```

**NATS Events:**
- `subscription.created` - New subscription notifications
- `subscription.status.changed` - Status change notifications
- `billing.invoice.ready` - Invoice generation complete
- `billing.payment.required` - Payment action required

#### 3. Payment & Financial Events

**Kafka Events:**
```go
type PaymentEvent struct {
    BaseEvent
    PaymentEventType   string             `json:"payment_event_type"`
    PaymentId          string             `json:"payment_id"`
    SubscriptionId     string             `json:"subscription_id,omitempty"`
    CustomerId         string             `json:"customer_id"`
    InvoiceId          string             `json:"invoice_id,omitempty"`
    Payment            entities.Payment   `json:"payment"`
    Amount             int64              `json:"amount"`
    Currency           string             `json:"currency"`
    PaymentMethod      string             `json:"payment_method"`
    PaymentStatus      string             `json:"payment_status"`
    ProcessorResponse  map[string]string  `json:"processor_response,omitempty"`
    FailureReason      string             `json:"failure_reason,omitempty"`
}

type RefundEvent struct {
    BaseEvent
    RefundEventType    string            `json:"refund_event_type"`
    RefundId           string            `json:"refund_id"`
    PaymentId          string            `json:"payment_id"`
    SubscriptionId     string            `json:"subscription_id,omitempty"`
    CustomerId         string            `json:"customer_id"`
    Amount             int64             `json:"amount"`
    Currency           string            `json:"currency"`
    RefundReason       string            `json:"refund_reason"`
    RefundStatus       string            `json:"refund_status"`
    ProcessorResponse  map[string]string `json:"processor_response,omitempty"`
}
```

**NATS Events:**
- `payment.processing` - Payment in progress
- `payment.success` - Payment completed successfully
- `payment.failed` - Payment failed notification
- `refund.processed` - Refund completion notification

#### 4. Customer Events

**Kafka Events:**
```go
type CustomerEvent struct {
    BaseEvent
    CustomerEventType  string              `json:"customer_event_type"`
    CustomerId         string              `json:"customer_id"`
    Customer           entities.Customer   `json:"customer"`
    PreviousEmail      string              `json:"previous_email,omitempty"`
    NewEmail           string              `json:"new_email,omitempty"`
    ProfileChanges     map[string]string   `json:"profile_changes,omitempty"`
}
```

**NATS Events:**
- `customer.created` - New customer welcome
- `customer.updated` - Profile updates
- `customer.payment_method.updated` - Payment method changes

#### 5. Product & Pricing Events

**Kafka Events:**
```go
type ProductEvent struct {
    BaseEvent
    ProductEventType   string            `json:"product_event_type"`
    ProductId          string            `json:"product_id"`
    Product            entities.Product  `json:"product"`
    PreviousState      *entities.Product `json:"previous_state,omitempty"`
}

type PriceEvent struct {
    BaseEvent
    PriceEventType     string          `json:"price_event_type"`
    PriceId            string          `json:"price_id"`
    ProductId          string          `json:"product_id"`
    Price              entities.Price  `json:"price"`
    PreviousState      *entities.Price `json:"previous_state,omitempty"`
}
```

**NATS Events:**
- `product.created` - New product available
- `price.updated` - Price changes
- `variant.created` - New variant available

#### 6. Dunning & Recovery Events

**Kafka Events:**
```go
type DunningEvent struct {
    BaseEvent
    DunningEventType   string                     `json:"dunning_event_type"`
    DunningCampaignId  string                     `json:"dunning_campaign_id"`
    SubscriptionId     string                     `json:"subscription_id"`
    CustomerId         string                     `json:"customer_id"`
    PaymentId          string                     `json:"payment_id,omitempty"`
    AttemptNumber      int                        `json:"attempt_number"`
    AttemptResult      string                     `json:"attempt_result"`
    NextAttemptDate    *time.Time                 `json:"next_attempt_date,omitempty"`
    CampaignStatus     string                     `json:"campaign_status"`
    CommunicationType  string                     `json:"communication_type,omitempty"`
    RecoveryAmount     int64                      `json:"recovery_amount,omitempty"`
}
```

**NATS Events:**
- `dunning.attempt.started` - Recovery attempt initiated
- `dunning.payment.recovered` - Payment successfully recovered
- `dunning.campaign.failed` - Final failure notification

### Event Type Mapping

#### Complete Event List with Publisher Assignment

```go
// Event Type Constants
const (
    // Usage Events (Kafka + NATS)
    UsageRecorded         = "usage.recorded"         // Both
    UsageBatchRecorded    = "usage.batch.recorded"   // Kafka only
    UsageAggregated       = "usage.aggregated"       // Kafka only
    UsageThresholdWarning = "usage.threshold.warning" // NATS only
    UsageThresholdExceeded = "usage.threshold.exceeded" // NATS only
    
    // Billing Events (Kafka + NATS)
    BillingInvoiceCreated    = "billing.invoice.created"    // Both
    BillingInvoicePaid       = "billing.invoice.paid"       // Both
    BillingInvoiceOverdue    = "billing.invoice.overdue"    // Both
    BillingPaymentRequired   = "billing.payment.required"   // NATS only
    BillingAmountCalculated  = "billing.amount.calculated"  // Kafka only
    
    // Subscription Events (Kafka + NATS)
    SubscriptionCreated     = "subscription.created"      // Both
    SubscriptionActivated   = "subscription.activated"    // Both
    SubscriptionPaused      = "subscription.paused"       // Both
    SubscriptionResumed     = "subscription.resumed"      // Both
    SubscriptionCancelled   = "subscription.cancelled"    // Both
    SubscriptionExpired     = "subscription.expired"      // Both
    SubscriptionPlanChanged = "subscription.plan.changed" // Both
    
    // Payment Events (Kafka + NATS)
    PaymentCreated     = "payment.created"      // Both
    PaymentProcessing  = "payment.processing"   // NATS only
    PaymentSucceeded   = "payment.succeeded"    // Both
    PaymentFailed      = "payment.failed"       // Both
    PaymentRefunded    = "payment.refunded"     // Both
    
    // Customer Events (Kafka + NATS)
    CustomerCreated          = "customer.created"            // Both
    CustomerUpdated          = "customer.updated"            // Both
    CustomerPaymentMethodUpdated = "customer.payment_method.updated" // Both
    CustomerDeleted          = "customer.deleted"            // Kafka only
    
    // Product Events (Kafka + NATS)
    ProductCreated = "product.created"   // Both
    ProductUpdated = "product.updated"   // Both
    ProductDeleted = "product.deleted"   // Kafka only
    PriceCreated   = "price.created"     // Both
    PriceUpdated   = "price.updated"     // Both
    PriceDeleted   = "price.deleted"     // Kafka only
    
    // Dunning Events (Kafka + NATS)
    DunningCampaignStarted     = "dunning.campaign.started"     // Both
    DunningAttemptStarted      = "dunning.attempt.started"      // NATS only
    DunningPaymentRecovered    = "dunning.payment.recovered"    // Both
    DunningCampaignFailed      = "dunning.campaign.failed"      // Both
    DunningSubscriptionSuspended = "dunning.subscription.suspended" // Both
    
    // Audit Events (Kafka only)
    AuditUserAction      = "audit.user.action"       // Kafka only
    AuditSystemAction    = "audit.system.action"     // Kafka only
    AuditDataChange      = "audit.data.change"       // Kafka only
    AuditSecurityEvent   = "audit.security.event"    // Kafka only
)
```

### Publisher Selection Rules

#### Kafka Events (Durable)
- **Compliance Requirements**: All billing, payment, and financial events
- **Audit Trail**: Customer changes, subscription modifications, refunds
- **Analytics**: Usage aggregations, billing calculations, business metrics
- **Event Sourcing**: State-changing events that need to be replayed
- **Regulatory**: Events required for tax reporting, compliance audits

#### NATS Events (Real-time)
- **UI Updates**: Real-time dashboard updates, progress indicators
- **Immediate Notifications**: Payment processing status, threshold alerts
- **User Experience**: Immediate feedback for user actions
- **System Alerts**: Operational notifications, system health
- **Temporary States**: Processing status, intermediate notifications

#### Both Publishers
- **Critical Business Events**: Subscription creation, payment success/failure
- **Customer-Facing Events**: Invoice creation, subscription changes
- **Operational Events**: Events that need both audit trail and real-time notification

## Kafka Configuration

### Topics
- `gphq.usage.recorded` - Usage recording and metering events
- `gphq.billing.events` - Billing, invoicing, and subscription events
- `gphq.payment.events` - Payment processing and refund events
- `gphq.customer.events` - Customer lifecycle and profile events
- `gphq.audit.events` - Audit trail and compliance events

### Partitioning Strategy
- **Partition Key**: `orgId` for tenant isolation
- **Replication Factor**: 3 for production
- **Retention**: 7 days for events, 30 days for audit logs

### Producer Configuration
```go
saramaConfig := sarama.NewConfig()
saramaConfig.Producer.Return.Successes = true
saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
saramaConfig.Producer.Retry.Max = 3
saramaConfig.Producer.Partitioner = sarama.NewHashPartitioner
```

## Service Integration Pattern

### Dual Publishing
```go
// 1. Save to database
usageRecord, err := s.repository.Create(ctx, usageRecord)
if err != nil {
    return err
}

// 2. Publish durable event (Kafka)
durableEvent := events.NewUsageRecordedEvent(orgId, usageRecord)
if err := s.durablePublisher.PublishUsageEvent(ctx, durableEvent); err != nil {
    s.logger.Warn("Failed to publish durable event", "error", err)
    // Don't fail operation - this is async
}

// 3. Publish notification (NATS)
if err := s.notificationPublisher.Publish(orgId, "usage.recorded", usageRecord); err != nil {
    s.logger.Warn("Failed to publish notification", "error", err)
}
```

## Error Handling Strategy

### Failure Modes
1. **Database Success + Kafka Failure**: Log warning, continue
2. **Database Success + NATS Failure**: Log warning, continue
3. **Database Failure**: Return error, no publishing
4. **Kafka Unavailable**: Graceful degradation, notification only

### Retry Strategy
- **Kafka**: Built-in Sarama retry (3 attempts)
- **NATS**: Single attempt (real-time nature)
- **Circuit Breaker**: Consider for Kafka if needed

## Testing Strategy

### Unit Tests
- Interface mocking for both publishers
- Event serialization/deserialization
- Factory function validation

### Integration Tests
- Embedded Kafka for testing
- NATS server integration
- Dual-publishing scenarios

### Performance Tests
- Throughput benchmarks
- Latency measurements
- Memory usage profiling

## Migration Strategy

### Phase 1: Non-Breaking Changes
1. Add new interfaces alongside existing
2. Implement Kafka publisher
3. Update DI configuration

### Phase 2: Service Updates
1. Update services one by one
2. Maintain existing NATS functionality
3. Add Kafka publishing where needed

### Phase 3: Interface Rename
1. Rename PubSub → NotificationPublisher
2. Update all references
3. Maintain backward compatibility

## Monitoring & Observability

### Logging
- Structured logging for all events
- Error tracking with context
- Performance measurements

### Health Checks
- Kafka broker connectivity
- NATS server health
- Publisher availability

## Security Considerations

### Access Control
- Kafka ACLs for topic access
- NATS subject-based permissions
- Service-to-service authentication

### Data Protection
- Event payload encryption (if needed)
- PII handling in events
- Audit log retention

## Future Enhancements

### Event Sourcing
- Event store implementation
- Event replay capabilities
- Snapshot generation

### Schema Evolution
- Event versioning strategy
- Schema registry integration
- Backward compatibility

### Advanced Features
- Dead letter queues
- Event deduplication
- Cross-region replication

## Success Criteria

1. **Functionality**: Both publishers work independently
2. **Performance**: No degradation in existing NATS performance
3. **Reliability**: Kafka events are durable and recoverable
4. **Maintainability**: Clear separation of concerns
5. **Testability**: Comprehensive test coverage
6. **Compatibility**: No breaking changes to existing code

## Implementation Phases

- **Phase 1**: Interface refactoring and domain events
- **Phase 2**: Kafka implementation and configuration
- **Phase 3**: Service integration and testing
- **Phase 4**: Documentation and performance optimization

## Dependencies

### External Libraries
- `github.com/IBM/sarama` - Kafka client
- `github.com/nats-io/nats.go` - NATS client (existing)
- `go.uber.org/fx` - Dependency injection (existing)

### Infrastructure
- Kafka cluster (development/staging/production)
- NATS server (existing)
- Monitoring tools (Prometheus, Grafana)

## Risk Assessment

### Technical Risks
- **Kafka Complexity**: Mitigation through proper configuration
- **Dual Publishing**: Potential inconsistency, handle gracefully
- **Performance Impact**: Monitor and optimize

### Operational Risks
- **Kafka Maintenance**: Requires operational expertise
- **Data Loss**: Proper replication and monitoring
- **Debugging**: Enhanced logging and tracing

This specification provides a comprehensive roadmap for implementing the dual-publisher system while maintaining system reliability and clean architecture principles.