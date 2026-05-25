package service

import (
	"context"
	"encoding/json"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// ReportEventBridge subscribes to operational pubsub topics and upserts the
// corresponding entity into the reporting database. The nightly
// ProcessDailyMetrics cron aggregates the resulting rows; any event missed
// while the subscriber is down will be reflected on the next event for that
// entity.
//
// Topics with no reporting table (orders, products, prices, variants, dunning)
// are intentionally not subscribed — add a subscription here when a matching
// Upsert lands on ReportRepository.
type ReportEventBridge struct {
	logger     port.Logger
	pubsub     port.PubSub
	reportRepo port.ReportRepository
}

func NewReportEventBridge(
	logger port.Logger,
	pubsub port.PubSub,
	reportRepo port.ReportRepository,
) *ReportEventBridge {
	b := &ReportEventBridge{
		logger:     logger,
		pubsub:     pubsub,
		reportRepo: reportRepo,
	}

	patterns := []string{
		"subscription.>",
		"payment.>",
		"customer.>",
		"refund.>",
	}
	for _, p := range patterns {
		logger.Debugf("[ReportEventBridge] subscribing to [%s]", p)
		if _, err := pubsub.Subscribe(p, b.Handle); err != nil {
			logger.Error("Failed to subscribe to topic", "topic", p, "error", err.Error())
			panic(err)
		}
	}
	return b
}

func (b *ReportEventBridge) Handle(topic string, data []byte) {
	var envelope port.PubSubPayload
	if err := json.Unmarshal(data, &envelope); err != nil {
		b.logger.Error("[ReportEventBridge] failed to unmarshal envelope", "topic", topic, "error", err.Error())
		return
	}

	payloadBytes, err := json.Marshal(envelope.Data)
	if err != nil {
		b.logger.Error("[ReportEventBridge] failed to marshal envelope data", "topic", topic, "error", err.Error())
		return
	}

	ctx := context.Background()

	switch topic {
	case port.TopicSubscriptionCreated,
		port.TopicSubscriptionPaused,
		port.TopicSubscriptionActivated,
		port.TopicSubscriptionResumed,
		port.TopicSubscriptionCancelled,
		port.TopicSubscriptionUnpaid,
		port.TopicSubscriptionExpired,
		port.TopicSubscriptionCompleted,
		port.TopicSubscriptionPastDue,
		port.TopicSubscriptionBillingAnchorChanged:
		var sub domain.Subscription
		if err := json.Unmarshal(payloadBytes, &sub); err != nil {
			b.logger.Error("[ReportEventBridge] failed to decode subscription", "topic", topic, "error", err.Error())
			return
		}
		if err := b.reportRepo.UpsertSubscription(ctx, sub); err != nil {
			b.logger.Error("[ReportEventBridge] failed to upsert subscription", "topic", topic, "error", err.Error())
		}

	case port.TopicSubscriptionPaymentChargeSuccess:
		var evt port.SubscriptionPaymentChargeSuccessEvent
		if err := json.Unmarshal(payloadBytes, &evt); err != nil {
			b.logger.Error("[ReportEventBridge] failed to decode charge-success event", "topic", topic, "error", err.Error())
			return
		}
		if err := b.reportRepo.UpsertPayment(ctx, evt.Payment); err != nil {
			b.logger.Error("[ReportEventBridge] failed to upsert payment", "topic", topic, "error", err.Error())
		}

	case port.TopicPaymentCreated, port.TopicPaymentUpdated, port.TopicPaymentFailed:
		var payment domain.Payment
		if err := json.Unmarshal(payloadBytes, &payment); err != nil {
			b.logger.Error("[ReportEventBridge] failed to decode payment", "topic", topic, "error", err.Error())
			return
		}
		if err := b.reportRepo.UpsertPayment(ctx, payment); err != nil {
			b.logger.Error("[ReportEventBridge] failed to upsert payment", "topic", topic, "error", err.Error())
		}

	case port.TopicCustomerCreated:
		var customer domain.Customer
		if err := json.Unmarshal(payloadBytes, &customer); err != nil {
			b.logger.Error("[ReportEventBridge] failed to decode customer", "topic", topic, "error", err.Error())
			return
		}
		if err := b.reportRepo.UpsertCustomer(ctx, customer); err != nil {
			b.logger.Error("[ReportEventBridge] failed to upsert customer", "topic", topic, "error", err.Error())
		}

	default:
		// payment_method.*, subscription.workflow.*, subscription.renewal_reminder,
		// and any other unrouted topic in the subscribed namespaces.
	}
}
