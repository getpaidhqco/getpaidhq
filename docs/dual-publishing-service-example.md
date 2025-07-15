---
title: Dual Publishing Service Integration Example
description: Complete example of how to integrate services with both Kafka and NATS publishers for dual event publishing
---

# Dual Publishing Service Integration Example

This document provides a complete example of how to integrate application services with the dual publishing system (Kafka + NATS).

## Service Constructor Pattern

### Before (Single Publisher)
```go
type UsageRecordingService struct {
    usageRecordRepo repositories.UsageRecordRepository
    pubsub          events.NotificationPublisher  // Old single publisher
    logger          logger.Logger
}

func NewUsageRecordingService(
    usageRecordRepo repositories.UsageRecordRepository,
    pubsub events.NotificationPublisher,
    logger logger.Logger,
) interfaces.UsageRecordingService {
    return &UsageRecordingService{
        usageRecordRepo: usageRecordRepo,
        pubsub:          pubsub,
        logger:          logger,
    }
}
```

### After (Dual Publishers)
```go
type UsageRecordingService struct {
    usageRecordRepo        repositories.UsageRecordRepository
    durablePublisher       events.DurableEventPublisher     // Kafka for audit
    notificationPublisher  events.NotificationPublisher     // NATS for real-time
    logger                 logger.Logger
}

func NewUsageRecordingService(
    usageRecordRepo repositories.UsageRecordRepository,
    durablePublisher events.DurableEventPublisher,
    notificationPublisher events.NotificationPublisher,
    logger logger.Logger,
) interfaces.UsageRecordingService {
    return &UsageRecordingService{
        usageRecordRepo:       usageRecordRepo,
        durablePublisher:      durablePublisher,
        notificationPublisher: notificationPublisher,
        logger:                logger,
    }
}
```

## Dual Publishing Implementation

### Complete Usage Recording Example

```go
func (s *UsageRecordingService) RecordUsage(
    ctx context.Context,
    orgId string,
    input dto.RecordUsageInput,
) (entities.UsageRecord, error) {
    // 1. Execute business logic
    usageRecord, err := s.createAndValidateUsageRecord(ctx, orgId, input)
    if err != nil {
        return entities.UsageRecord{}, err
    }

    // 2. Persist to database
    savedRecord, err := s.usageRecordRepo.Create(ctx, usageRecord)
    if err != nil {
        return entities.UsageRecord{}, err
    }

    // 3. Publish durable event (Kafka) - for audit trail
    if err := s.publishDurableEvent(ctx, orgId, savedRecord); err != nil {
        s.logger.Warn("Failed to publish durable usage event", 
            "error", err,
            "usage_record_id", savedRecord.Id,
            "org_id", orgId)
        // Don't fail the operation - this is async
    }

    // 4. Publish notification (NATS) - for real-time UI updates
    if err := s.publishNotification(ctx, orgId, savedRecord); err != nil {
        s.logger.Warn("Failed to publish usage notification",
            "error", err,
            "usage_record_id", savedRecord.Id,
            "org_id", orgId)
        // Don't fail the operation - this is async
    }

    return savedRecord, nil
}

// publishDurableEvent publishes structured event to Kafka for audit trail
func (s *UsageRecordingService) publishDurableEvent(
    ctx context.Context,
    orgId string,
    usageRecord entities.UsageRecord,
) error {
    // Create structured domain event
    event := events.NewUsageRecordedEvent(orgId, usageRecord)
    
    // Publish to Kafka for audit trail
    return s.durablePublisher.PublishUsageEvent(ctx, event)
}

// publishNotification publishes minimal event to NATS for real-time updates
func (s *UsageRecordingService) publishNotification(
    ctx context.Context,
    orgId string,
    usageRecord entities.UsageRecord,
) error {
    // Create minimal notification payload
    notification := map[string]interface{}{
        "usage_record_id":   usageRecord.Id,
        "subscription_id":   usageRecord.SubscriptionId,
        "customer_id":       usageRecord.CustomerId,
        "quantity":          usageRecord.Quantity,
        "metric_name":       string(usageRecord.UsageType),
        "billing_period":    usageRecord.BillingPeriod,
        "timestamp":         time.Now().UTC(),
        "display_message":   fmt.Sprintf("%.2f %s recorded", usageRecord.Quantity, usageRecord.UsageType),
    }
    
    // Publish to NATS for real-time notifications
    return s.notificationPublisher.Publish(orgId, "usage.recorded", notification)
}
```

## Payment Service Example

### Payment Processing with Dual Publishing

```go
func (s *PaymentService) ProcessPayment(
    ctx context.Context,
    orgId string,
    paymentId string,
) error {
    // 1. Execute payment processing
    payment, err := s.chargeCustomer(ctx, paymentId)
    if err != nil {
        return err
    }

    // 2. Update payment status in database
    if err := s.paymentRepo.Update(ctx, payment); err != nil {
        return err
    }

    // 3. Determine event type based on payment status
    var eventType string
    switch payment.Status {
    case payments.PaymentStatusCompleted:
        eventType = events.PaymentSucceeded
    case payments.PaymentStatusFailed:
        eventType = events.PaymentFailed
    default:
        eventType = events.PaymentCreated
    }

    // 4. Publish durable event (Kafka)
    if err := s.publishPaymentEvent(ctx, orgId, eventType, payment); err != nil {
        s.logger.Warn("Failed to publish payment audit event", "error", err)
    }

    // 5. Publish notification (NATS)
    if err := s.publishPaymentNotification(ctx, orgId, eventType, payment); err != nil {
        s.logger.Warn("Failed to publish payment notification", "error", err)
    }

    return nil
}

func (s *PaymentService) publishPaymentEvent(
    ctx context.Context,
    orgId string,
    eventType string,
    payment entities.Payment,
) error {
    // Create complete audit event
    event := events.NewPaymentEvent(orgId, eventType, payment)
    return s.durablePublisher.PublishPaymentEvent(ctx, event)
}

func (s *PaymentService) publishPaymentNotification(
    ctx context.Context,
    orgId string,
    eventType string,
    payment entities.Payment,
) error {
    // Create user-friendly notification
    var message, status string
    switch eventType {
    case events.PaymentSucceeded:
        message = "Payment processed successfully"
        status = "success"
    case events.PaymentFailed:
        message = "Payment failed"
        status = "failed"
    default:
        message = "Payment is being processed"
        status = "processing"
    }

    notification := map[string]interface{}{
        "payment_id":     payment.Id,
        "customer_id":    payment.OrderId, // Derive customer from order
        "amount":         payment.Amount,
        "currency":       payment.Currency,
        "status":         status,
        "message":        message,
        "display_amount": formatCurrency(payment.Amount, payment.Currency),
        "timestamp":      time.Now().UTC(),
    }

    // Map to appropriate NATS topic
    var topic string
    switch status {
    case "success":
        topic = "payment.success"
    case "failed":
        topic = "payment.failed"
    default:
        topic = "payment.processing"
    }

    return s.notificationPublisher.Publish(orgId, topic, notification)
}
```

## Subscription Service Example

### Subscription State Changes

```go
func (s *SubscriptionService) CancelSubscription(
    ctx context.Context,
    orgId string,
    subscriptionId string,
    reason string,
) error {
    // 1. Load current subscription
    subscription, err := s.subscriptionRepo.FindById(ctx, orgId, subscriptionId)
    if err != nil {
        return err
    }

    previousStatus := subscription.Status

    // 2. Execute cancellation business logic
    subscription.Status = entities.SubscriptionStatusCancelled
    subscription.CancelledAt = time.Now().UTC()
    subscription.CancellationReason = reason

    // 3. Save to database
    if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
        return err
    }

    // 4. Publish durable event (Kafka) - complete audit trail
    if err := s.publishSubscriptionEvent(ctx, orgId, subscription, previousStatus, reason); err != nil {
        s.logger.Warn("Failed to publish subscription audit event", "error", err)
    }

    // 5. Publish notification (NATS) - immediate user feedback
    if err := s.publishSubscriptionNotification(ctx, orgId, subscription); err != nil {
        s.logger.Warn("Failed to publish subscription notification", "error", err)
    }

    return nil
}

func (s *SubscriptionService) publishSubscriptionEvent(
    ctx context.Context,
    orgId string,
    subscription entities.Subscription,
    previousStatus string,
    reason string,
) error {
    event := events.NewSubscriptionEvent(
        orgId,
        events.SubscriptionCancelled,
        subscription,
        previousStatus,
        string(subscription.Status),
        reason,
    )
    
    return s.durablePublisher.PublishSubscriptionEvent(ctx, event)
}

func (s *SubscriptionService) publishSubscriptionNotification(
    ctx context.Context,
    orgId string,
    subscription entities.Subscription,
) error {
    notification := map[string]interface{}{
        "subscription_id": subscription.Id,
        "customer_id":     subscription.CustomerId,
        "status":          "cancelled",
        "message":         "Subscription cancelled successfully",
        "effective_date":  subscription.CancelledAt.Format("2006-01-02"),
        "access_until":    subscription.CurrentPeriodEnd.Format("2006-01-02"),
        "plan_name":       subscription.ProductName, // If available
    }

    return s.notificationPublisher.Publish(orgId, "subscription.cancelled", notification)
}
```

## Error Handling Patterns

### Graceful Degradation

```go
func (s *BaseService) publishEvents(
    ctx context.Context,
    orgId string,
    durableEvent interface{},
    notificationTopic string,
    notificationData interface{},
) {
    // Publish durable event (Kafka) - critical for audit
    if err := s.publishDurableEventSafely(ctx, durableEvent); err != nil {
        s.logger.Error("CRITICAL: Failed to publish audit event",
            "error", err,
            "org_id", orgId,
            "event_type", getDurableEventType(durableEvent))
        // Consider alerting operations team for critical events
    }

    // Publish notification (NATS) - best effort
    if err := s.notificationPublisher.Publish(orgId, notificationTopic, notificationData); err != nil {
        s.logger.Warn("Failed to publish notification",
            "error", err,
            "org_id", orgId,
            "topic", notificationTopic)
        // Notification failure doesn't affect business operation
    }
}

func (s *BaseService) publishDurableEventSafely(ctx context.Context, event interface{}) error {
    switch e := event.(type) {
    case events.UsageRecordedEvent:
        return s.durablePublisher.PublishUsageEvent(ctx, e)
    case events.PaymentEvent:
        return s.durablePublisher.PublishPaymentEvent(ctx, e)
    case events.SubscriptionEvent:
        return s.durablePublisher.PublishSubscriptionEvent(ctx, e)
    default:
        return fmt.Errorf("unsupported event type: %T", event)
    }
}
```

## Testing Patterns

### Service Testing with Mock Publishers

```go
func TestUsageRecordingService_RecordUsage(t *testing.T) {
    // Setup
    usageRepo := &mocks.MockUsageRecordRepository{}
    durablePublisher := &mocks.MockDurableEventPublisher{}
    notificationPublisher := &mocks.MockNotificationPublisher{}
    logger := &mocks.MockLogger{}

    service := NewUsageRecordingService(
        usageRepo,
        durablePublisher,
        notificationPublisher,
        logger,
    )

    // Mock expectations
    usageRepo.On("Create", mock.Anything, mock.Anything).Return(expectedUsageRecord, nil)
    durablePublisher.On("PublishUsageEvent", mock.Anything, mock.Anything).Return(nil)
    notificationPublisher.On("Publish", "org_123", "usage.recorded", mock.Anything).Return(nil)

    // Execute
    result, err := service.RecordUsage(ctx, "org_123", input)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expectedUsageRecord, result)
    
    // Verify both publishers were called
    durablePublisher.AssertExpectations(t)
    notificationPublisher.AssertExpectations(t)
}
```

## Migration Strategy

### Phase 1: Add Publishers to Constructor
1. Add both publishers to service constructors
2. Update dependency injection configuration
3. Deploy without publishing events

### Phase 2: Add Dual Publishing
1. Implement dual publishing in one service at a time
2. Test both publishers work independently
3. Monitor logs for any publishing failures

### Phase 3: Optimize
1. Add batch publishing for high-volume events
2. Implement circuit breakers if needed
3. Add metrics and monitoring

## Best Practices

### Do's ✅
- Always publish durable events for audit-critical operations
- Use structured domain events for Kafka
- Use simple, user-friendly payloads for NATS
- Log publishing failures but don't fail operations
- Use appropriate event types for each publisher

### Don'ts ❌
- Don't fail business operations due to publishing failures
- Don't publish sensitive data in NATS notifications
- Don't use the same payload for both publishers
- Don't publish every minor state change to Kafka
- Don't ignore publishing errors completely

### Event Selection Guide

**Publish to Kafka when:**
- Financial transaction occurs
- Customer data changes
- Subscription state changes
- Compliance audit required
- Event sourcing needed

**Publish to NATS when:**
- User needs immediate feedback
- Dashboard needs real-time updates
- Temporary status changes
- Progress indicators needed
- System health updates

**Publish to both when:**
- Critical business events
- Customer-facing operations
- State changes affecting multiple systems
- Events requiring both audit and user experience

This pattern ensures that your services maintain both comprehensive audit trails and excellent user experience through the dual publishing architecture.