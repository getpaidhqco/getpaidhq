package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
)

// UsageEventConsumer handles consumption of raw usage events from Kafka
// Implements interfaces.Consumer
type UsageEventConsumer struct {
	brokers    []string
	topic      string
	groupID    string
	repository repositories.UsageEventRepository
	logger     logger.Logger
	consumer   sarama.ConsumerGroup
	ready      chan bool
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// usageEventHandler implements sarama.ConsumerGroupHandler
type usageEventHandler struct {
	consumer *UsageEventConsumer
	ctx      context.Context
	ready    chan bool
}

// NewUsageEventConsumer creates a new usage event consumer
func NewUsageEventConsumer(
	brokers []string,
	topic string,
	groupID string,
	repository repositories.UsageEventRepository,
	logger logger.Logger,
) interfaces.Consumer {
	return &UsageEventConsumer{
		brokers:    brokers,
		topic:      topic,
		groupID:    groupID,
		repository: repository,
		logger:     logger,
		ready:      make(chan bool),
	}
}

// Start begins consuming messages from the Kafka topic
func (c *UsageEventConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting usage event consumer", "topic", c.topic, "group", c.groupID)

	// Create a new context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// Create Sarama config
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true

	// Create consumer group
	consumer, err := sarama.NewConsumerGroup(c.brokers, c.groupID, config)
	if err != nil {
		c.logger.Error("Failed to create consumer group", "error", err)
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	c.consumer = consumer

	// Track errors
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for err := range consumer.Errors() {
			c.logger.Error("Consumer group error", "error", err)
		}
	}()

	// Create consumer handler
	handler := &usageEventHandler{
		consumer: c,
		ctx:      ctx,
		ready:    c.ready,
	}

	// Start consuming
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			// Check if context was cancelled
			if ctx.Err() != nil {
				c.logger.Info("Context cancelled, stopping consumer")
				return
			}

			// Consume from topic
			if err := consumer.Consume(ctx, []string{c.topic}, handler); err != nil {
				c.logger.Error("Error from consumer", "error", err)
			}

			// Check if context was cancelled
			if ctx.Err() != nil {
				return
			}

			// Mark the consumer as ready
			select {
			case <-c.ready:
				// Continue to next loop iteration
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait until the consumer is ready
	<-c.ready
	c.logger.Info("Usage event consumer ready")

	return nil
}

// Stop gracefully shuts down the consumer
func (c *UsageEventConsumer) Stop() error {
	c.logger.Info("Stopping usage event consumer")

	if c.cancel != nil {
		c.cancel()
	}

	if c.consumer != nil {
		if err := c.consumer.Close(); err != nil {
			c.logger.Error("Error closing consumer", "error", err)
			return err
		}
	}

	c.wg.Wait()
	c.logger.Info("Usage event consumer stopped")
	return nil
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *usageEventHandler) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(h.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *usageEventHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *usageEventHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if h.ctx.Err() != nil {
			return nil
		}

		// Process the message
		if err := h.processMessage(h.ctx, message); err != nil {
			h.consumer.logger.Error("Error processing message", "error", err)
			// Continue processing other messages even if one fails
		}

		// Mark the message as processed
		session.MarkMessage(message, "")
	}
	return nil
}

// processMessage processes a single Kafka message
func (h *usageEventHandler) processMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// Deserialize the event
	var event events.RawUsageRecordedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal usage event: %w", err)
	}

	// Convert event to domain entity
	usageEvent := entities.UsageEvent{
		OrgId:              event.OrgId,
		Id:                 event.EventId,
		SubscriptionId:     event.SubscriptionId,
		SubscriptionItemId: event.SubscriptionItemId,
		MeterId:            event.MeterId,
		// CloudEvents fields
		SpecVersion: "1.0",
		Type:        event.EventType,
		EventId:     event.EventId,
		Time:        event.Timestamp,
		Source:      event.Metadata["source"],
		Subject:     event.SubscriptionItemId,
		Data:        event.Data.(map[string]interface{}),
		// Audit
		ReceivedAt: event.ReceivedAt,
	}

	// Store the entity using the repository interface
	if err := h.consumer.repository.Create(ctx, usageEvent); err != nil {
		return fmt.Errorf("failed to store usage event: %w", err)
	}

	h.consumer.logger.Info("Successfully processed usage event", 
		"event_id", event.EventId, 
		"subscription_item_id", event.SubscriptionItemId)

	return nil
}
