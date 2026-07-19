package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// Outbox is a port.PubSub whose Publish writes the encoded envelope to the
// outbox table instead of the broker; the relay delivers it after commit.
// Subscribe and Close delegate to the real broker adapter.
type Outbox struct {
	repo     port.OutboxRepository
	delegate port.PubSub
}

func NewOutbox(repo port.OutboxRepository, delegate port.PubSub) *Outbox {
	return &Outbox{repo: repo, delegate: delegate}
}

func (o *Outbox) Publish(ctx context.Context, orgId string, topic string, message any) error {
	createdAt := time.Now().UTC()
	envelope := domain.PubSubPayload{
		Id:        lib.GenerateId("evt"),
		OrgId:     orgId,
		Topic:     topic,
		Data:      message,
		CreatedAt: createdAt,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal pubsub payload for %q: %w", topic, err)
	}
	return o.repo.Create(ctx, domain.OutboxEvent{
		EventId:   envelope.Id,
		OrgId:     orgId,
		Topic:     topic,
		Payload:   data,
		CreatedAt: createdAt,
	})
}

func (o *Outbox) Subscribe(topic string, handler func(topic string, data []byte)) (port.PubSubSubscription, error) {
	return o.delegate.Subscribe(topic, handler)
}

func (o *Outbox) Close() error {
	return o.delegate.Close()
}
