package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/IBM/sarama"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"sync"
	"time"
)

// KafkaPublisher implements the DurableEventPublisher interface using Kafka
type KafkaPublisher struct {
	producer sarama.SyncProducer
	config   Config
	logger   logger.Logger
	mu       sync.Mutex
}

// NewKafkaPublisher creates a new Kafka publisher
func NewKafkaPublisher(config Config, logger logger.Logger) (events.DurableEventPublisher, error) {
	saramaConfig := NewSaramaConfig(config)

	producer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		logger.Error("Failed to create Kafka producer", "error", err)
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &KafkaPublisher{
		producer: producer,
		config:   config,
		logger:   logger,
	}, nil
}

// Health checks the health of the Kafka publisher
func (k *KafkaPublisher) Health() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.producer == nil {
		return fmt.Errorf("kafka producer is not initialized")
	}

	// For a SyncProducer, if it's initialized, it should be healthy
	// The producer will automatically reconnect if the connection is lost
	return nil
}

// Close closes the Kafka producer
func (k *KafkaPublisher) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.producer != nil {
		return k.producer.Close()
	}
	return nil
}

// reconnectProducer attempts to reconnect to Kafka by creating a new producer
func (k *KafkaPublisher) reconnectProducer() error {
	// Close the existing producer if it exists
	if k.producer != nil {
		_ = k.producer.Close()
	}

	// Create a new producer
	saramaConfig := NewSaramaConfig(k.config)
	producer, err := sarama.NewSyncProducer(k.config.Brokers, saramaConfig)
	if err != nil {
		k.logger.Error("Failed to reconnect to Kafka", "error", err)
		return fmt.Errorf("failed to reconnect to Kafka: %w", err)
	}

	k.producer = producer
	k.logger.Info("Successfully reconnected to Kafka")
	return nil
}

// PublishUsageEvent publishes a usage event to Kafka
func (k *KafkaPublisher) PublishUsageEvent(ctx context.Context, event events.RawUsageRecordedEvent) error {
	return k.publishEvent(ctx, events.TopicUsageEvents, event.OrgId, event)
}

// PublishBillingEvent publishes a billing event to Kafka
func (k *KafkaPublisher) PublishBillingEvent(ctx context.Context, event events.BillingEvent) error {
	return k.publishEvent(ctx, events.TopicBillingEvents, event.OrgId, event)
}

// PublishPaymentEvent publishes a payment event to Kafka
func (k *KafkaPublisher) PublishPaymentEvent(ctx context.Context, event events.PaymentEvent) error {
	return k.publishEvent(ctx, events.TopicPaymentEvents, event.OrgId, event)
}

// PublishSubscriptionEvent publishes a subscription event to Kafka
func (k *KafkaPublisher) PublishSubscriptionEvent(ctx context.Context, event events.SubscriptionEvent) error {
	return k.publishEvent(ctx, events.TopicBillingEvents, event.OrgId, event)
}

// PublishCustomerEvent publishes a customer event to Kafka
func (k *KafkaPublisher) PublishCustomerEvent(ctx context.Context, event events.CustomerEvent) error {
	return k.publishEvent(ctx, events.TopicCustomerEvents, event.OrgId, event)
}

// PublishInvoiceEvent publishes an invoice event to Kafka
func (k *KafkaPublisher) PublishInvoiceEvent(ctx context.Context, event events.InvoiceEvent) error {
	return k.publishEvent(ctx, events.TopicBillingEvents, event.OrgId, event)
}

// PublishRefundEvent publishes a refund event to Kafka
func (k *KafkaPublisher) PublishRefundEvent(ctx context.Context, event events.RefundEvent) error {
	return k.publishEvent(ctx, events.TopicPaymentEvents, event.OrgId, event)
}

// PublishProductEvent publishes a product event to Kafka
func (k *KafkaPublisher) PublishProductEvent(ctx context.Context, event events.ProductEvent) error {
	return k.publishEvent(ctx, events.TopicBillingEvents, event.OrgId, event)
}

// PublishPriceEvent publishes a price event to Kafka
func (k *KafkaPublisher) PublishPriceEvent(ctx context.Context, event events.PriceEvent) error {
	return k.publishEvent(ctx, events.TopicBillingEvents, event.OrgId, event)
}

// PublishDunningEvent publishes a dunning event to Kafka
func (k *KafkaPublisher) PublishDunningEvent(ctx context.Context, event events.DunningEvent) error {
	return k.publishEvent(ctx, events.TopicBillingEvents, event.OrgId, event)
}

// PublishUsageBatch publishes a batch of usage events to Kafka
func (k *KafkaPublisher) PublishUsageBatch(ctx context.Context, usageEvents []events.RawUsageRecordedEvent) error {
	if len(usageEvents) == 0 {
		return nil
	}

	// Group events by orgId
	eventsByOrg := make(map[string][]events.RawUsageRecordedEvent)
	for _, event := range usageEvents {
		eventsByOrg[event.OrgId] = append(eventsByOrg[event.OrgId], event)
	}

	// Publish events for each orgId
	for orgId, orgEvents := range eventsByOrg {
		if err := k.publishBatch(ctx, events.TopicUsageEvents, orgId, orgEvents); err != nil {
			return err
		}
	}

	return nil
}

// PublishEventBatch publishes a batch of base events to Kafka
func (k *KafkaPublisher) PublishEventBatch(ctx context.Context, usageEvents []events.BaseEvent) error {
	if len(usageEvents) == 0 {
		return nil
	}

	// Group events by topic and orgId
	eventsByTopicAndOrg := make(map[string]map[string][]events.BaseEvent)
	for _, event := range usageEvents {
		topic := k.getTopicForEventType(event.EventType)
		if _, ok := eventsByTopicAndOrg[topic]; !ok {
			eventsByTopicAndOrg[topic] = make(map[string][]events.BaseEvent)
		}
		eventsByTopicAndOrg[topic][event.OrgId] = append(eventsByTopicAndOrg[topic][event.OrgId], event)
	}

	// Publish events for each topic and orgId
	for topic, eventsByOrg := range eventsByTopicAndOrg {
		for orgId, orgEvents := range eventsByOrg {
			if err := k.publishBatch(ctx, topic, orgId, orgEvents); err != nil {
				return err
			}
		}
	}

	return nil
}

// publishEvent publishes a single event to Kafka
func (k *KafkaPublisher) publishEvent(ctx context.Context, topic, orgId string, event interface{}) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Convert event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create Kafka message
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(orgId), // Use orgId as partition key
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("org_id"),
				Value: []byte(orgId),
			},
			{
				Key:   []byte("timestamp"),
				Value: []byte(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	}

	// Send message to Kafka
	partition, offset, err := k.producer.SendMessage(msg)
	if err != nil {
		k.logger.Error("Failed to publish event to Kafka", "error", err, "topic", topic)

		// Attempt to reconnect if the producer is closed or connection is lost
		if err == sarama.ErrClosedClient || err == sarama.ErrOutOfBrokers || err == sarama.ErrNotConnected {
			k.logger.Info("Attempting to reconnect to Kafka", "brokers", k.config.Brokers)

			// Reconnect to Kafka
			if reconnectErr := k.reconnectProducer(); reconnectErr != nil {
				return fmt.Errorf("failed to publish event to Kafka and reconnection failed: %w", err)
			}

			// Retry sending the message
			partition, offset, err = k.producer.SendMessage(msg)
			if err != nil {
				k.logger.Error("Failed to publish event to Kafka after reconnection", "error", err)
				return fmt.Errorf("failed to publish event to Kafka after reconnection: %w", err)
			}
		} else {
			return fmt.Errorf("failed to publish event to Kafka: %w", err)
		}
	}

	k.logger.Debug("Published event to Kafka", "topic", topic, "partition", partition, "offset", offset)
	return nil
}

// publishBatch publishes a batch of events to Kafka
func (k *KafkaPublisher) publishBatch(ctx context.Context, topic, orgId string, events interface{}) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Convert events to JSON
	data, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// Create Kafka message
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(orgId), // Use orgId as partition key
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("org_id"),
				Value: []byte(orgId),
			},
			{
				Key:   []byte("batch"),
				Value: []byte("true"),
			},
			{
				Key:   []byte("timestamp"),
				Value: []byte(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	}

	// Send message to Kafka
	partition, offset, err := k.producer.SendMessage(msg)
	if err != nil {
		k.logger.Error("Failed to publish batch to Kafka", "error", err, "topic", topic)

		// Attempt to reconnect if the producer is closed or connection is lost
		if err == sarama.ErrClosedClient || err == sarama.ErrOutOfBrokers || err == sarama.ErrNotConnected {
			k.logger.Info("Attempting to reconnect to Kafka", "brokers", k.config.Brokers)

			// Reconnect to Kafka
			if reconnectErr := k.reconnectProducer(); reconnectErr != nil {
				return fmt.Errorf("failed to publish batch to Kafka and reconnection failed: %w", err)
			}

			// Retry sending the message
			partition, offset, err = k.producer.SendMessage(msg)
			if err != nil {
				k.logger.Error("Failed to publish batch to Kafka after reconnection", "error", err)
				return fmt.Errorf("failed to publish batch to Kafka after reconnection: %w", err)
			}
		} else {
			return fmt.Errorf("failed to publish batch to Kafka: %w", err)
		}
	}

	k.logger.Debug("Published batch to Kafka", "topic", topic, "partition", partition, "offset", offset)
	return nil
}

// getTopicForEventType returns the appropriate Kafka topic for the given event type
func (k *KafkaPublisher) getTopicForEventType(eventType string) string {
	switch {
	case eventType == events.UsageRecorded || eventType == events.UsageBatchRecorded || eventType == events.UsageAggregated:
		return events.TopicUsageEvents
	case eventType == events.PaymentCreated || eventType == events.PaymentSucceeded || eventType == events.PaymentFailed || eventType == events.PaymentRefunded:
		return events.TopicPaymentEvents
	case eventType == events.CustomerCreated || eventType == events.CustomerUpdated || eventType == events.CustomerPaymentMethodUpdated || eventType == events.CustomerDeleted:
		return events.TopicCustomerEvents
	case eventType == events.AuditUserAction || eventType == events.AuditSystemAction || eventType == events.AuditDataChange || eventType == events.AuditSecurityEvent:
		return events.TopicAuditEvents
	default:
		return events.TopicBillingEvents
	}
}
