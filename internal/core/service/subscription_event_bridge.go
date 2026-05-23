package service

import (
	"context"
	"encoding/json"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// SubscriptionEventBridge fans pubsub "subscription.*" topics into the
// workflow engine as update events on the per-subscription durable runner.
//
// Both engine adapters used to reimplement this independently inside their
// own pubsub subscriber. Lifting it here keeps the adapters engine-shaped
// only — they don't need to know about pubsub topic conventions or how to
// decode envelopes.
type SubscriptionEventBridge struct {
	engine port.Engine
	pubsub port.PubSub
	logger port.Logger
}

func NewSubscriptionEventBridge(
	engine port.Engine,
	pubsub port.PubSub,
	logger port.Logger,
) *SubscriptionEventBridge {
	b := &SubscriptionEventBridge{
		engine: engine,
		pubsub: pubsub,
		logger: logger,
	}

	logger.Debugf("[SubscriptionEventBridge] Subscribing to subscription.*")
	if _, err := pubsub.Subscribe("subscription.*", b.Handle); err != nil {
		logger.Error("Failed to subscribe to subscription.* topic", "error", err.Error())
		panic(err)
	}
	return b
}

// Handle decodes the pubsub envelope and forwards relevant transitions to the
// engine. Topics that have no engine-side observer are dropped.
func (b *SubscriptionEventBridge) Handle(topic string, data []byte) {
	b.logger.Infof("[SubscriptionEventBridge] received [%s]", topic)

	var envelope port.PubSubPayload
	if err := json.Unmarshal(data, &envelope); err != nil {
		b.logger.Error("Failed to unmarshal envelope", "error", err.Error())
		return
	}

	payloadBytes, err := json.Marshal(envelope.Data)
	if err != nil {
		b.logger.Error("Failed to marshal subscription data", "error", err.Error())
		return
	}
	var sub domain.Subscription
	if err := json.Unmarshal(payloadBytes, &sub); err != nil {
		b.logger.Error("Failed to unmarshal subscription", "error", err.Error())
		return
	}

	switch topic {
	case port.TopicSubscriptionPaused:
		if err := b.engine.UpdateSubscriptionWorkflow(context.Background(), topic, sub); err != nil {
			b.logger.Error("Failed to forward subscription event", "error", err.Error(), "topic", topic)
		}
	default:
		b.logger.Infof("[SubscriptionEventBridge] no engine handler for %s", topic)
	}
}
