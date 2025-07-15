package kafka

import (
	"context"
	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"testing"
	"time"
)

// mockLogger is a simple implementation of the logger.Logger interface for testing
type mockLogger struct{}

func (m mockLogger) Debug(msg string, args ...interface{}) {}
func (m mockLogger) Info(msg string, args ...interface{})  {}
func (m mockLogger) Warn(msg string, args ...interface{})  {}
func (m mockLogger) Error(msg string, args ...interface{}) {}

func TestKafkaPublisher_PublishUsageEvent(t *testing.T) {
	// Create a mock Sarama config
	config := DefaultConfig()
	
	// Create a mock producer
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()
	
	// Create a Kafka publisher with the mock producer
	publisher := &KafkaPublisher{
		producer: mockProducer,
		config:   config,
		logger:   mockLogger{},
	}
	
	// Create a test event
	event := events.UsageRecordedEvent{
		BaseEvent: events.BaseEvent{
			EventId:          "evt_test",
			EventType:        events.UsageRecorded,
			OrgId:            "test-org",
			AggregateId:      "test-subscription",
			AggregateType:    "subscription",
			AggregateVersion: 1,
			Timestamp:        time.Now().UTC(),
			Metadata:         map[string]string{"test": "metadata"},
		},
		SubscriptionId:     "test-subscription",
		SubscriptionItemId: "test-item",
		CustomerId:         "test-customer",
		UsageRecord:        nil,
		MetricName:         "api-calls",
		Quantity:           100,
		UnitPrice:          10,
		BillingPeriod:      "2023-01",
	}
	
	// Publish the event
	err := publisher.PublishUsageEvent(context.Background(), event)
	
	// Assert that there was no error
	assert.NoError(t, err)
	
	// Close the publisher
	err = publisher.Close()
	assert.NoError(t, err)
}

func TestKafkaPublisher_PublishBillingEvent(t *testing.T) {
	// Create a mock Sarama config
	config := DefaultConfig()
	
	// Create a mock producer
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()
	
	// Create a Kafka publisher with the mock producer
	publisher := &KafkaPublisher{
		producer: mockProducer,
		config:   config,
		logger:   mockLogger{},
	}
	
	// Create a test event
	event := events.BillingEvent{
		BaseEvent: events.BaseEvent{
			EventId:          "evt_test",
			EventType:        events.BillingInvoiceCreated,
			OrgId:            "test-org",
			AggregateId:      "test-invoice",
			AggregateType:    "invoice",
			AggregateVersion: 1,
			Timestamp:        time.Now().UTC(),
			Metadata:         map[string]string{"test": "metadata"},
		},
		BillingEventType:   events.BillingInvoiceCreated,
		SubscriptionId:     "test-subscription",
		CustomerId:         "test-customer",
		InvoiceId:          "test-invoice",
		Amount:             1000,
		Currency:           "USD",
		BillingPeriodStart: time.Now().UTC().AddDate(0, -1, 0),
		BillingPeriodEnd:   time.Now().UTC(),
		TaxAmount:          100,
		DiscountAmount:     50,
	}
	
	// Publish the event
	err := publisher.PublishBillingEvent(context.Background(), event)
	
	// Assert that there was no error
	assert.NoError(t, err)
	
	// Close the publisher
	err = publisher.Close()
	assert.NoError(t, err)
}

func TestKafkaPublisher_PublishUsageBatch(t *testing.T) {
	// Create a mock Sarama config
	config := DefaultConfig()
	
	// Create a mock producer
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()
	
	// Create a Kafka publisher with the mock producer
	publisher := &KafkaPublisher{
		producer: mockProducer,
		config:   config,
		logger:   mockLogger{},
	}
	
	// Create test events
	events := []events.UsageRecordedEvent{
		{
			BaseEvent: events.BaseEvent{
				EventId:          "evt_test1",
				EventType:        events.UsageRecorded,
				OrgId:            "test-org",
				AggregateId:      "test-subscription1",
				AggregateType:    "subscription",
				AggregateVersion: 1,
				Timestamp:        time.Now().UTC(),
				Metadata:         map[string]string{"test": "metadata1"},
			},
			SubscriptionId:     "test-subscription1",
			SubscriptionItemId: "test-item1",
			CustomerId:         "test-customer1",
			UsageRecord:        nil,
			MetricName:         "api-calls",
			Quantity:           100,
			UnitPrice:          10,
			BillingPeriod:      "2023-01",
		},
		{
			BaseEvent: events.BaseEvent{
				EventId:          "evt_test2",
				EventType:        events.UsageRecorded,
				OrgId:            "test-org",
				AggregateId:      "test-subscription2",
				AggregateType:    "subscription",
				AggregateVersion: 1,
				Timestamp:        time.Now().UTC(),
				Metadata:         map[string]string{"test": "metadata2"},
			},
			SubscriptionId:     "test-subscription2",
			SubscriptionItemId: "test-item2",
			CustomerId:         "test-customer2",
			UsageRecord:        nil,
			MetricName:         "api-calls",
			Quantity:           200,
			UnitPrice:          20,
			BillingPeriod:      "2023-01",
		},
	}
	
	// Publish the events
	err := publisher.PublishUsageBatch(context.Background(), events)
	
	// Assert that there was no error
	assert.NoError(t, err)
	
	// Close the publisher
	err = publisher.Close()
	assert.NoError(t, err)
}