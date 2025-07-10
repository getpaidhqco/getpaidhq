package fixtures

import (
	"fmt"
	"time"

	"payloop/internal/domain/entities"
)

// UsageEventBuilder helps create test usage events
type UsageEventBuilder struct {
	event entities.UsageEvent
}

// NewUsageEventBuilder creates a new usage event builder with defaults
func NewUsageEventBuilder(orgId, subscriptionId, subscriptionItemId string) *UsageEventBuilder {
	now := time.Now()
	return &UsageEventBuilder{
		event: entities.UsageEvent{
			Id:                 fmt.Sprintf("evt_%d", now.UnixNano()),
			OrgId:              orgId,
			SubscriptionId:     subscriptionId,
			SubscriptionItemId: subscriptionItemId,
			MeterId:            "meter_default",
			SpecVersion:        "1.0",
			Type:               "usage.recorded",
			EventId:            fmt.Sprintf("event_%d", now.UnixNano()),
			Time:               now,
			Source:             "test",
			Subject:            subscriptionItemId,
			Data: map[string]interface{}{
				"quantity": float64(1),
			},
			ReceivedAt: now,
		},
	}
}

// WithQuantity sets the quantity for the usage event
func (b *UsageEventBuilder) WithQuantity(quantity float64) *UsageEventBuilder {
	b.event.Data["quantity"] = quantity
	return b
}

// WithTransactionValue sets the transaction value for percentage-based billing
func (b *UsageEventBuilder) WithTransactionValue(value float64) *UsageEventBuilder {
	b.event.Data["transaction_value"] = value
	return b
}

// WithTime sets the event time
func (b *UsageEventBuilder) WithTime(t time.Time) *UsageEventBuilder {
	b.event.Time = t
	return b
}

// WithMeterId sets the meter ID
func (b *UsageEventBuilder) WithMeterId(meterId string) *UsageEventBuilder {
	b.event.MeterId = meterId
	return b
}

// WithData sets custom data fields
func (b *UsageEventBuilder) WithData(data map[string]interface{}) *UsageEventBuilder {
	b.event.Data = data
	return b
}

// Build returns the constructed usage event
func (b *UsageEventBuilder) Build() entities.UsageEvent {
	return b.event
}

// CreateUsageEventsForPeriod creates a series of usage events across a time period
func CreateUsageEventsForPeriod(
	orgId, subscriptionId, subscriptionItemId string,
	startTime, endTime time.Time,
	dailyQuantity float64,
) []entities.UsageEvent {
	var events []entities.UsageEvent
	
	current := startTime
	for current.Before(endTime) {
		event := NewUsageEventBuilder(orgId, subscriptionId, subscriptionItemId).
			WithTime(current).
			WithQuantity(dailyQuantity).
			Build()
		events = append(events, event)
		current = current.Add(24 * time.Hour)
	}
	
	return events
}

// CreateTransactionEvents creates events with transaction values for percentage billing
func CreateTransactionEvents(
	orgId, subscriptionId, subscriptionItemId string,
	transactions []struct {
		Time  time.Time
		Value float64
	},
) []entities.UsageEvent {
	var events []entities.UsageEvent
	
	for _, tx := range transactions {
		event := NewUsageEventBuilder(orgId, subscriptionId, subscriptionItemId).
			WithTime(tx.Time).
			WithQuantity(1).
			WithTransactionValue(tx.Value).
			Build()
		events = append(events, event)
	}
	
	return events
}

// CreateMaxUsageEvents creates events for testing max aggregation
func CreateMaxUsageEvents(
	orgId, subscriptionId, subscriptionItemId string,
	period time.Time,
	quantities []float64,
) []entities.UsageEvent {
	var events []entities.UsageEvent
	
	for i, qty := range quantities {
		event := NewUsageEventBuilder(orgId, subscriptionId, subscriptionItemId).
			WithTime(period.Add(time.Duration(i) * time.Hour)).
			WithQuantity(qty).
			Build()
		events = append(events, event)
	}
	
	return events
}