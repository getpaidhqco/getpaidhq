package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
)

// UsageEventConsumer consumes usage events from NATS
type UsageEventConsumer struct {
	conn         *nats.Conn
	topic        string
	subscription *nats.Subscription
	repository   repositories.UsageEventRepository
	logger       logger.Logger
	mu           sync.Mutex
	isRunning    bool
	stopCh       chan struct{}
}

// NewUsageEventConsumer creates a new usage event consumer
func NewUsageEventConsumer(
	topic string,
	repository repositories.UsageEventRepository,
	logger logger.Logger,
) interfaces.Consumer {
	return &UsageEventConsumer{
		topic:      topic,
		repository: repository,
		logger:     logger,
		stopCh:     make(chan struct{}),
	}
}

// Start starts the consumer
func (c *UsageEventConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return nil
	}
	c.isRunning = true
	c.mu.Unlock()

	c.logger.Info("Starting NATS usage event consumer", "topic", c.topic)

	// Connect to NATS server
	var err error
	c.conn, err = nats.Connect(nats.DefaultURL)
	if err != nil {
		c.isRunning = false
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Subscribe to the topic
	c.subscription, err = c.conn.Subscribe(c.topic, func(msg *nats.Msg) {
		if err := c.processMessage(ctx, msg); err != nil {
			c.logger.Error("Failed to process message", "error", err)
		}
	})

	if err != nil {
		c.isRunning = false
		c.conn.Close()
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	// Wait for context cancellation or stop signal
	select {
	case <-ctx.Done():
		return c.Stop()
	case <-c.stopCh:
		return nil
	}
}

// Stop stops the consumer
func (c *UsageEventConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return nil
	}

	c.logger.Info("Stopping NATS usage event consumer")

	// Unsubscribe and close connection
	if c.subscription != nil {
		if err := c.subscription.Unsubscribe(); err != nil {
			c.logger.Error("Failed to unsubscribe", "error", err)
		}
	}

	if c.conn != nil {
		c.conn.Close()
	}

	close(c.stopCh)
	c.isRunning = false
	return nil
}

// processMessage processes a message from NATS
func (c *UsageEventConsumer) processMessage(ctx context.Context, msg *nats.Msg) error {
	c.logger.Debug("Processing message", "topic", c.topic)

	// Parse the message payload
	var payload struct {
		Id        string                     `json:"id"`
		OrgId     string                     `json:"org_id"`
		Topic     string                     `json:"topic"`
		Data      events.RawUsageRecordedEvent `json:"data"`
		CreatedAt time.Time                  `json:"created_at"`
	}

	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Extract the usage event
	event := payload.Data

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
	if err := c.repository.Create(ctx, usageEvent); err != nil {
		return fmt.Errorf("failed to store usage event: %w", err)
	}

	c.logger.Debug("Successfully processed usage event", 
		"event_id", event.EventId,
		"org_id", event.OrgId,
		"meter_id", event.MeterId)

	return nil
}
