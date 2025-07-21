package kafka

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"sync"
	"time"
)

// ReportingEventConsumer handles consumption of domain events for reporting database synchronization
type ReportingEventConsumer struct {
	brokers           []string
	topics            []string
	groupID           string
	reportRepository  repositories.ReportRepository
	logger            logger.Logger
	consumer          sarama.ConsumerGroup
	ready             chan bool
	cancel            context.CancelFunc
	processingTimeout time.Duration
}

// consumerGroupHandler implements the sarama.ConsumerGroupHandler interface
type reportingConsumerGroupHandler struct {
	consumer *ReportingEventConsumer
	ctx      context.Context
	ready    chan bool
	once     sync.Once
}

// NewReportingEventConsumer creates a new reporting event consumer
func NewReportingEventConsumer(
	brokers []string,
	topics []string,
	groupID string,
	reportRepository repositories.ReportRepository,
	logger logger.Logger,
) (*ReportingEventConsumer, error) {
	return &ReportingEventConsumer{
		brokers:           brokers,
		topics:            topics,
		groupID:           groupID,
		reportRepository:  reportRepository,
		logger:            logger,
		ready:             make(chan bool),
		processingTimeout: 30 * time.Second,
	}, nil
}

// Start begins consuming messages from the Kafka topics
func (c *ReportingEventConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting reporting event consumer", "topics", c.topics, "group", c.groupID)

	// Create a new context with cancellation
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// Create Kafka consumer
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true

	// Create consumer group
	consumer, err := sarama.NewConsumerGroup(c.brokers, c.groupID, config)
	if err != nil {
		c.logger.Error("Failed to create consumer group", "error", err)
		return err
	}
	c.consumer = consumer

	// Start consuming in a goroutine
	go func() {
		handler := &reportingConsumerGroupHandler{
			consumer: c,
			ctx:      ctx,
			ready:    c.ready,
		}

		for {
			// Check if context is cancelled
			if ctx.Err() != nil {
				c.logger.Info("Context cancelled, stopping consumer")
				return
			}

			// Consume from topics
			if err := consumer.Consume(ctx, c.topics, handler); err != nil {
				c.logger.Error("Error from consumer", "error", err)
				time.Sleep(time.Second) // Wait before retrying
			}

			// Check if context is cancelled
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// Wait for consumer to be ready
	<-c.ready
	c.logger.Info("Reporting event consumer ready")

	return nil
}

// Stop stops the consumer
func (c *ReportingEventConsumer) Stop() error {
	c.logger.Info("Stopping reporting event consumer")
	if c.cancel != nil {
		c.cancel()
	}

	if c.consumer != nil {
		if err := c.consumer.Close(); err != nil {
			c.logger.Error("Failed to close consumer", "error", err)
			return err
		}
	}

	c.logger.Info("Reporting event consumer stopped")
	return nil
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *reportingConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	h.once.Do(func() {
		close(h.ready)
	})
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *reportingConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *reportingConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if h.ctx.Err() != nil {
			return nil
		}

		// Process the message
		h.consumer.processMessage(h.ctx, message)

		// Mark the message as processed
		session.MarkMessage(message, "")
	}
	return nil
}

// processMessage processes a single Kafka message
func (c *ReportingEventConsumer) processMessage(ctx context.Context, message *sarama.ConsumerMessage) {
	c.logger.Debug("Processing message", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)

	// Create a context with timeout for processing
	ctx, cancel := context.WithTimeout(ctx, c.processingTimeout)
	defer cancel()

	// Determine the event type from the message
	var baseEvent events.BaseEvent
	if err := json.Unmarshal(message.Value, &baseEvent); err != nil {
		c.logger.Error("Failed to unmarshal base event", "error", err)
		return
	}

	// Process different event types
	switch baseEvent.EventType {
	case events.SubscriptionCreated, events.SubscriptionActivated, events.SubscriptionPaused, 
	     events.SubscriptionResumed, events.SubscriptionCancelled, events.SubscriptionExpired, 
	     events.SubscriptionPlanChanged:
		c.processSubscriptionEvent(ctx, message.Value)

	case events.PaymentCreated, events.PaymentSucceeded, events.PaymentFailed:
		c.processPaymentEvent(ctx, message.Value)

	case events.PaymentRefunded:
		c.processRefundEvent(ctx, message.Value)

	case events.CustomerCreated, events.CustomerUpdated, events.CustomerDeleted:
		c.processCustomerEvent(ctx, message.Value)

	case events.BillingInvoiceCreated, events.BillingInvoicePaid, events.BillingInvoiceOverdue:
		c.processInvoiceEvent(ctx, message.Value)

	case events.ProductCreated, events.ProductUpdated, events.ProductDeleted:
		c.processProductEvent(ctx, message.Value)

	case events.PriceCreated, events.PriceUpdated, events.PriceDeleted:
		c.processPriceEvent(ctx, message.Value)

	default:
		c.logger.Debug("Ignoring event type", "eventType", baseEvent.EventType)
	}
}

// processSubscriptionEvent processes a subscription event
func (c *ReportingEventConsumer) processSubscriptionEvent(ctx context.Context, data []byte) {
	var event events.SubscriptionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal subscription event", "error", err)
		return
	}

	c.logger.Debug("Processing subscription event", "eventType", event.EventType, "subscriptionId", event.SubscriptionId)

	// Upsert the subscription in the reporting database
	if err := c.reportRepository.UpsertSubscription(ctx, event.Subscription); err != nil {
		c.logger.Error("Failed to upsert subscription", "error", err)
		return
	}
}

// processPaymentEvent processes a payment event
func (c *ReportingEventConsumer) processPaymentEvent(ctx context.Context, data []byte) {
	var event events.PaymentEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal payment event", "error", err)
		return
	}

	c.logger.Debug("Processing payment event", "eventType", event.EventType, "paymentId", event.PaymentId)

	// Upsert the payment in the reporting database
	if err := c.reportRepository.UpsertPayment(ctx, event.Payment); err != nil {
		c.logger.Error("Failed to upsert payment", "error", err)
		return
	}
}

// processCustomerEvent processes a customer event
func (c *ReportingEventConsumer) processCustomerEvent(ctx context.Context, data []byte) {
	var event events.CustomerEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal customer event", "error", err)
		return
	}

	c.logger.Debug("Processing customer event", "eventType", event.EventType, "customerId", event.CustomerId)

	// Upsert the customer in the reporting database
	if err := c.reportRepository.UpsertCustomer(ctx, event.Customer); err != nil {
		c.logger.Error("Failed to upsert customer", "error", err)
		return
	}
}

// processInvoiceEvent processes an invoice event
func (c *ReportingEventConsumer) processInvoiceEvent(ctx context.Context, data []byte) {
	var event events.InvoiceEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal invoice event", "error", err)
		return
	}

	c.logger.Debug("Processing invoice event", "eventType", event.EventType, "invoiceId", event.InvoiceId)

	// Upsert the invoice in the reporting database
	// Note: The report repository might not have a method for upserting invoices yet
	// This would need to be added if it doesn't exist
}

// processProductEvent processes a product event
func (c *ReportingEventConsumer) processProductEvent(ctx context.Context, data []byte) {
	var event events.ProductEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal product event", "error", err)
		return
	}

	c.logger.Debug("Processing product event", "eventType", event.EventType, "productId", event.ProductId)

	// Upsert the product in the reporting database
	// Note: The report repository might not have a method for upserting products yet
	// This would need to be added if it doesn't exist
}

// processPriceEvent processes a price event
func (c *ReportingEventConsumer) processPriceEvent(ctx context.Context, data []byte) {
	var event events.PriceEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal price event", "error", err)
		return
	}

	c.logger.Debug("Processing price event", "eventType", event.EventType, "priceId", event.PriceId)

	// Upsert the price in the reporting database
	// Note: The report repository might not have a method for upserting prices yet
	// This would need to be added if it doesn't exist
}

// processRefundEvent processes a refund event
func (c *ReportingEventConsumer) processRefundEvent(ctx context.Context, data []byte) {
	var event events.RefundEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal refund event", "error", err)
		return
	}

	c.logger.Debug("Processing refund event", "eventType", event.EventType, "refundId", event.RefundId)

	// Upsert the refund in the reporting database
	if err := c.reportRepository.UpsertRefund(ctx, entities.Refund{
		OrgId:     event.OrgId,
		Id:        event.RefundId,
		PaymentId: event.PaymentId,
		Amount:    event.Amount,
		Currency:  event.Currency,
		Reason:    event.RefundReason,
		Status:    entities.RefundStatus(event.RefundStatus),
		CreatedAt: event.Timestamp,
		UpdatedAt: event.Timestamp,
	}); err != nil {
		c.logger.Error("Failed to upsert refund", "error", err)
		return
	}
}
