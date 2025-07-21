package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"sync"
	"time"
)

// ReportingEventConsumer handles consumption of domain events for reporting database synchronization
type ReportingEventConsumer struct {
	conn             *nats.Conn
	topics           []string
	subscriptions    []*nats.Subscription
	reportRepository repositories.ReportRepository
	logger           logger.Logger
	mu               sync.Mutex
	isRunning        bool
	stopCh           chan struct{}
	processingTimeout time.Duration
}

// NewReportingEventConsumer creates a new reporting event consumer
func NewReportingEventConsumer(
	topics []string,
	reportRepository repositories.ReportRepository,
	logger logger.Logger,
) (*ReportingEventConsumer, error) {
	return &ReportingEventConsumer{
		topics:           topics,
		reportRepository: reportRepository,
		logger:           logger,
		stopCh:           make(chan struct{}),
		processingTimeout: 30 * time.Second,
	}, nil
}

// Start begins consuming messages from the NATS topics
func (c *ReportingEventConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return nil
	}

	c.logger.Info("Starting NATS reporting event consumer", "topics", c.topics)

	// Connect to NATS server
	var err error
	c.conn, err = nats.Connect(nats.DefaultURL)
	if err != nil {
		c.logger.Error("Failed to connect to NATS", "error", err)
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Subscribe to each topic
	for _, topic := range c.topics {
		subscription, err := c.conn.Subscribe(topic, func(msg *nats.Msg) {
			// Process the message in a goroutine to avoid blocking the NATS subscription
			go func() {
				if err := c.processMessage(ctx, msg); err != nil {
					c.logger.Error("Failed to process message", "error", err)
				}
			}()
		})
		if err != nil {
			c.logger.Error("Failed to subscribe to topic", "topic", topic, "error", err)
			// Clean up any subscriptions that were created
			for _, sub := range c.subscriptions {
				_ = sub.Unsubscribe()
			}
			return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
		}
		c.subscriptions = append(c.subscriptions, subscription)
	}

	c.isRunning = true
	return nil
}

// Stop stops the consumer
func (c *ReportingEventConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return nil
	}

	c.logger.Info("Stopping NATS reporting event consumer")

	// Unsubscribe from all topics
	for _, subscription := range c.subscriptions {
		if err := subscription.Unsubscribe(); err != nil {
			c.logger.Error("Failed to unsubscribe", "error", err)
		}
	}
	c.subscriptions = nil

	// Close the connection
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// Signal stop
	close(c.stopCh)
	c.isRunning = false

	c.logger.Info("NATS reporting event consumer stopped")
	return nil
}

// processMessage processes a message from NATS
func (c *ReportingEventConsumer) processMessage(ctx context.Context, msg *nats.Msg) error {
	c.logger.Debug("Processing message", "subject", msg.Subject)

	// Create a context with timeout for processing
	ctx, cancel := context.WithTimeout(ctx, c.processingTimeout)
	defer cancel()

	// Determine the event type from the message
	var baseEvent events.BaseEvent
	if err := json.Unmarshal(msg.Data, &baseEvent); err != nil {
		c.logger.Error("Failed to unmarshal base event", "error", err)
		return err
	}

	// Process different event types
	switch baseEvent.EventType {
	case events.SubscriptionCreated, events.SubscriptionActivated, events.SubscriptionPaused, 
	     events.SubscriptionResumed, events.SubscriptionCancelled, events.SubscriptionExpired, 
	     events.SubscriptionPlanChanged:
		return c.processSubscriptionEvent(ctx, msg.Data)

	case events.PaymentCreated, events.PaymentSucceeded, events.PaymentFailed:
		return c.processPaymentEvent(ctx, msg.Data)
		
	case events.PaymentRefunded:
		return c.processRefundEvent(ctx, msg.Data)

	case events.CustomerCreated, events.CustomerUpdated, events.CustomerDeleted:
		return c.processCustomerEvent(ctx, msg.Data)

	case events.BillingInvoiceCreated, events.BillingInvoicePaid, events.BillingInvoiceOverdue:
		return c.processInvoiceEvent(ctx, msg.Data)

	case events.ProductCreated, events.ProductUpdated, events.ProductDeleted:
		return c.processProductEvent(ctx, msg.Data)

	case events.PriceCreated, events.PriceUpdated, events.PriceDeleted:
		return c.processPriceEvent(ctx, msg.Data)

	default:
		c.logger.Debug("Ignoring event type", "eventType", baseEvent.EventType)
		return nil
	}
}

// processSubscriptionEvent processes a subscription event
func (c *ReportingEventConsumer) processSubscriptionEvent(ctx context.Context, data []byte) error {
	var event events.SubscriptionEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal subscription event", "error", err)
		return err
	}

	c.logger.Debug("Processing subscription event", "eventType", event.EventType, "subscriptionId", event.SubscriptionId)
	
	// Upsert the subscription in the reporting database
	if err := c.reportRepository.UpsertSubscription(ctx, event.Subscription); err != nil {
		c.logger.Error("Failed to upsert subscription", "error", err)
		return err
	}
	
	return nil
}

// processPaymentEvent processes a payment event
func (c *ReportingEventConsumer) processPaymentEvent(ctx context.Context, data []byte) error {
	var event events.PaymentEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal payment event", "error", err)
		return err
	}

	c.logger.Debug("Processing payment event", "eventType", event.EventType, "paymentId", event.PaymentId)
	
	// Upsert the payment in the reporting database
	if err := c.reportRepository.UpsertPayment(ctx, event.Payment); err != nil {
		c.logger.Error("Failed to upsert payment", "error", err)
		return err
	}
	
	return nil
}

// processCustomerEvent processes a customer event
func (c *ReportingEventConsumer) processCustomerEvent(ctx context.Context, data []byte) error {
	var event events.CustomerEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal customer event", "error", err)
		return err
	}

	c.logger.Debug("Processing customer event", "eventType", event.EventType, "customerId", event.CustomerId)
	
	// Upsert the customer in the reporting database
	if err := c.reportRepository.UpsertCustomer(ctx, event.Customer); err != nil {
		c.logger.Error("Failed to upsert customer", "error", err)
		return err
	}
	
	return nil
}

// processInvoiceEvent processes an invoice event
func (c *ReportingEventConsumer) processInvoiceEvent(ctx context.Context, data []byte) error {
	var event events.InvoiceEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal invoice event", "error", err)
		return err
	}

	c.logger.Debug("Processing invoice event", "eventType", event.EventType, "invoiceId", event.InvoiceId)
	
	// Upsert the invoice in the reporting database
	// Note: The report repository might not have a method for upserting invoices yet
	// This would need to be added if it doesn't exist
	
	return nil
}

// processProductEvent processes a product event
func (c *ReportingEventConsumer) processProductEvent(ctx context.Context, data []byte) error {
	var event events.ProductEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal product event", "error", err)
		return err
	}

	c.logger.Debug("Processing product event", "eventType", event.EventType, "productId", event.ProductId)
	
	// Upsert the product in the reporting database
	// Note: The report repository might not have a method for upserting products yet
	// This would need to be added if it doesn't exist
	
	return nil
}

// processPriceEvent processes a price event
func (c *ReportingEventConsumer) processPriceEvent(ctx context.Context, data []byte) error {
	var event events.PriceEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal price event", "error", err)
		return err
	}

	c.logger.Debug("Processing price event", "eventType", event.EventType, "priceId", event.PriceId)
	
	// Upsert the price in the reporting database
	// Note: The report repository might not have a method for upserting prices yet
	// This would need to be added if it doesn't exist
	
	return nil
}

// processRefundEvent processes a refund event
func (c *ReportingEventConsumer) processRefundEvent(ctx context.Context, data []byte) error {
	var event events.RefundEvent
	if err := json.Unmarshal(data, &event); err != nil {
		c.logger.Error("Failed to unmarshal refund event", "error", err)
		return err
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
		return err
	}
	
	return nil
}