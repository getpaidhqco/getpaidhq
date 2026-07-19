package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// OutboxPubSub is the default port.PubSub: Publish writes the encoded event
// envelope to the outbox instead of the broker, so a publish inside RunInTx
// is atomic with the business write and the relay delivers it after commit.
// Subscribe and Close delegate to the real broker adapter unchanged.
type OutboxPubSub struct {
	repo     port.OutboxRepository
	delegate port.PubSub
}

func NewOutboxPubSub(repo port.OutboxRepository, delegate port.PubSub) *OutboxPubSub {
	return &OutboxPubSub{repo: repo, delegate: delegate}
}

// Compile-time check that OutboxPubSub satisfies the port.
var _ port.PubSub = (*OutboxPubSub)(nil)

func (o *OutboxPubSub) Publish(ctx context.Context, orgId string, topic string, message any) error {
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

func (o *OutboxPubSub) Subscribe(topic string, handler func(topic string, data []byte)) (port.PubSubSubscription, error) {
	return o.delegate.Subscribe(topic, handler)
}

func (o *OutboxPubSub) Close() error {
	return o.delegate.Close()
}
