package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

// NatsDurablePublisher implements the DurableEventPublisher interface for NATS
type NatsDurablePublisher struct {
	conn   *nats.Conn
	logger logger.Logger
}

// NewNatsDurablePublisher creates a new NATS durable publisher
func NewNatsDurablePublisher(logger logger.Logger) (events.DurableEventPublisher, error) {
	// Connect to NATS server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NatsDurablePublisher{
		conn:   nc,
		logger: logger,
	}, nil
}

// PublishUsageEvent publishes a usage event to NATS
func (n *NatsDurablePublisher) PublishUsageEvent(ctx context.Context, event events.RawUsageRecordedEvent) error {
	return n.publishEvent(ctx, "gphq.usage.recorded", event.OrgId, event)
}

// PublishBillingEvent publishes a billing event to NATS
func (n *NatsDurablePublisher) PublishBillingEvent(ctx context.Context, event events.BillingEvent) error {
	return n.publishEvent(ctx, "gphq.billing.events", event.OrgId, event)
}

// PublishPaymentEvent publishes a payment event to NATS
func (n *NatsDurablePublisher) PublishPaymentEvent(ctx context.Context, event events.PaymentEvent) error {
	return n.publishEvent(ctx, "gphq.payment.events", event.OrgId, event)
}

// PublishSubscriptionEvent publishes a subscription event to NATS
func (n *NatsDurablePublisher) PublishSubscriptionEvent(ctx context.Context, event events.SubscriptionEvent) error {
	return n.publishEvent(ctx, "gphq.billing.events", event.OrgId, event)
}

// PublishCustomerEvent publishes a customer event to NATS
func (n *NatsDurablePublisher) PublishCustomerEvent(ctx context.Context, event events.CustomerEvent) error {
	return n.publishEvent(ctx, "gphq.customer.events", event.OrgId, event)
}

// PublishInvoiceEvent publishes an invoice event to NATS
func (n *NatsDurablePublisher) PublishInvoiceEvent(ctx context.Context, event events.InvoiceEvent) error {
	return n.publishEvent(ctx, "gphq.billing.events", event.OrgId, event)
}

// PublishRefundEvent publishes a refund event to NATS
func (n *NatsDurablePublisher) PublishRefundEvent(ctx context.Context, event events.RefundEvent) error {
	return n.publishEvent(ctx, "gphq.payment.events", event.OrgId, event)
}

// PublishProductEvent publishes a product event to NATS
func (n *NatsDurablePublisher) PublishProductEvent(ctx context.Context, event events.ProductEvent) error {
	return n.publishEvent(ctx, "gphq.billing.events", event.OrgId, event)
}

// PublishPriceEvent publishes a price event to NATS
func (n *NatsDurablePublisher) PublishPriceEvent(ctx context.Context, event events.PriceEvent) error {
	return n.publishEvent(ctx, "gphq.billing.events", event.OrgId, event)
}

// PublishDunningEvent publishes a dunning event to NATS
func (n *NatsDurablePublisher) PublishDunningEvent(ctx context.Context, event events.DunningEvent) error {
	return n.publishEvent(ctx, "gphq.billing.events", event.OrgId, event)
}

// PublishUsageBatch publishes a batch of usage events to NATS
func (n *NatsDurablePublisher) PublishUsageBatch(ctx context.Context, events []events.RawUsageRecordedEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Use the first event's OrgId for the batch
	orgId := events[0].OrgId
	return n.publishBatch(ctx, "gphq.usage.recorded", orgId, events)
}

// PublishEventBatch publishes a batch of events to NATS
func (n *NatsDurablePublisher) PublishEventBatch(ctx context.Context, events []events.BaseEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Use the first event's OrgId for the batch
	orgId := events[0].OrgId
	return n.publishBatch(ctx, n.getTopicForEventType(events[0].EventType), orgId, events)
}

// publishEvent publishes an event to NATS
func (n *NatsDurablePublisher) publishEvent(ctx context.Context, topic string, orgId string, event interface{}) error {
	// Create payload with metadata
	payload := struct {
		Id        string      `json:"id"`
		OrgId     string      `json:"org_id"`
		Topic     string      `json:"topic"`
		Data      interface{} `json:"data"`
		CreatedAt time.Time   `json:"created_at"`
	}{
		Id:        lib.GenerateId("evt"),
		OrgId:     orgId,
		Topic:     topic,
		Data:      event,
		CreatedAt: time.Now().UTC(),
	}

	// Marshal payload to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Log the event
	n.logger.Debug(fmt.Sprintf("[nats] publishing to topic [%s]", topic))

	// Publish to NATS
	err = n.conn.Publish(topic, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// publishBatch publishes a batch of events to NATS
func (n *NatsDurablePublisher) publishBatch(ctx context.Context, topic string, orgId string, events interface{}) error {
	// Create payload with metadata
	payload := struct {
		Id        string      `json:"id"`
		OrgId     string      `json:"org_id"`
		Topic     string      `json:"topic"`
		Data      interface{} `json:"data"`
		CreatedAt time.Time   `json:"created_at"`
	}{
		Id:        lib.GenerateId("evt"),
		OrgId:     orgId,
		Topic:     topic,
		Data:      events,
		CreatedAt: time.Now().UTC(),
	}

	// Marshal payload to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event batch: %w", err)
	}

	// Log the event
	n.logger.Debug(fmt.Sprintf("[nats] publishing batch to topic [%s]", topic))

	// Publish to NATS
	err = n.conn.Publish(topic, data)
	if err != nil {
		return fmt.Errorf("failed to publish event batch: %w", err)
	}

	return nil
}

// getTopicForEventType returns the appropriate topic for the given event type
func (n *NatsDurablePublisher) getTopicForEventType(eventType string) string {
	switch eventType {
	case events.UsageRecorded, events.UsageBatchRecorded, events.UsageAggregated:
		return "gphq.usage.recorded"
	case events.PaymentCreated, events.PaymentSucceeded, events.PaymentFailed, events.PaymentRefunded:
		return "gphq.payment.events"
	case events.CustomerCreated, events.CustomerUpdated, events.CustomerPaymentMethodUpdated, events.CustomerDeleted:
		return "gphq.customer.events"
	case events.AuditUserAction, events.AuditSystemAction, events.AuditDataChange, events.AuditSecurityEvent:
		return "gphq.audit.events"
	default:
		return "gphq.billing.events"
	}
}
