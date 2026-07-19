package port

import (
	"context"

	"getpaidhq/internal/core/domain"
)

// PubSub defines the interface for publish/subscribe messaging. Publish takes
// ctx so the outbox implementation can join the ambient transaction — an
// event published inside RunInTx commits and rolls back with the business
// write.
type PubSub interface {
	Publish(ctx context.Context, orgId string, topic string, message any) error
	Subscribe(topic string, handler func(topic string, data []byte)) (PubSubSubscription, error)
	// Close drains in-flight messages and releases the underlying connection
	// (and, for the embedded-server adapter, shuts the server down). Safe to
	// call once during graceful shutdown.
	Close() error
}

// PubSubPayload is an alias for the domain PubSubPayload type.
type PubSubPayload = domain.PubSubPayload

type PubSubSubscription interface {
	Unsubscribe() error
}

// RawPublisher publishes an already-encoded envelope to a topic. Implemented
// by the NATS adapter and used by the outbox relay, so stored envelopes go
// out verbatim instead of being wrapped a second time.
type RawPublisher interface {
	PublishPayload(topic string, data []byte) error
}
