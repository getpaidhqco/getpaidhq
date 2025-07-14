package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

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
	once     sync.Once
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

	// Set session timeout and heartbeat interval to ensure proper group membership
	config.Consumer.Group.Session.Timeout = 20 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 6 * time.Second

	// Enable auto commit to ensure offsets are committed regularly
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Offsets.AutoCommit.Interval = 5 * time.Second

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
		once:     sync.Once{},
	}

	// Start consuming
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		// Backoff for reconnection attempts
		backoff := time.Second
		maxBackoff := 30 * time.Second

		for {
			// Check if context was cancelled
			if ctx.Err() != nil {
				c.logger.Info("Context cancelled, stopping consumer")
				return
			}

			// Consume from topic
			c.logger.Info("Joining consumer group", "group", c.groupID, "topic", c.topic)
			if err := consumer.Consume(ctx, []string{c.topic}, handler); err != nil {
				if ctx.Err() != nil {
					// Context was cancelled, exit gracefully
					return
				}

				c.logger.Error("Error from consumer", "error", err, "backoff", backoff.String())

				// Wait before reconnecting with exponential backoff
				select {
				case <-time.After(backoff):
					// Double the backoff for next attempt, up to the maximum
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				case <-ctx.Done():
					return
				}

				continue
			}

			// Reset backoff on successful connection
			backoff = time.Second

			// Check if context was cancelled
			if ctx.Err() != nil {
				return
			}

			c.logger.Info("Consumer group session ended, rejoining")

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
func (h *usageEventHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.consumer.logger.Info("Consumer group session setup",
		"member_id", session.MemberID(),
		"generation_id", session.GenerationID())

	// Mark the consumer as ready - use sync.Once to ensure channel is only closed once
	h.once.Do(func() {
		close(h.ready)
	})
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *usageEventHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.consumer.logger.Info("Consumer group session cleanup",
		"member_id", session.MemberID(),
		"generation_id", session.GenerationID())

	// Create a new ready channel for the next session
	h.consumer.ready = make(chan bool)
	h.ready = h.consumer.ready
	h.once = sync.Once{}

	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *usageEventHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.consumer.logger.Info("Starting to consume from partition",
		"topic", claim.Topic(),
		"partition", claim.Partition(),
		"initial_offset", claim.InitialOffset(),
		"member_id", session.MemberID())

	// Use a separate context for message processing to ensure we can cancel it if needed
	ctx, cancel := context.WithCancel(h.ctx)
	defer cancel()

	// Watch for context cancellation
	go func() {
		<-h.ctx.Done()
		cancel()
	}()

	for message := range claim.Messages() {
		// Check if context was cancelled
		if ctx.Err() != nil {
			h.consumer.logger.Info("Context cancelled, stopping message consumption")
			return nil
		}

		h.consumer.logger.Debug("Received message",
			"topic", message.Topic,
			"partition", message.Partition,
			"offset", message.Offset)

		// Process the message
		if err := h.processMessage(ctx, message); err != nil {
			h.consumer.logger.Error("Error processing message",
				"error", err,
				"topic", message.Topic,
				"partition", message.Partition,
				"offset", message.Offset)
			// Continue processing other messages even if one fails
		}

		// Mark the message as processed
		session.MarkMessage(message, "")

		// Commit offsets regularly to avoid issues with rebalancing
		if message.Offset%1000 == 0 {
			session.Commit()
		}
	}

	h.consumer.logger.Info("Finished consuming from partition",
		"topic", claim.Topic(),
		"partition", claim.Partition(),
		"member_id", session.MemberID())

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
		OrgId:   event.OrgId,
		Id:      event.Id,
		MeterId: event.MeterId,
		// CloudEvents fields
		SpecVersion: "1.0",
		Type:        event.EventType,
		EventId:     event.EventId,
		Time:        event.Timestamp,
		Source:      event.Metadata["source"],
		Subject:     event.Subject,
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
		"subject", event.Metadata["subject"])

	return nil
}
