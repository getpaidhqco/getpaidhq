package port

import (
	"context"

	"getpaidhq/internal/core/domain"
)

// PubSub defines the interface for publish/subscribe messaging. Publish takes
// ctx so an event published inside RunInTx joins the ambient transaction.
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

// RawPublisher publishes an encoded event envelope verbatim, without wrapping
// it in a new envelope. The outbox relay delivers stored rows through this.
type RawPublisher interface {
	PublishPayload(topic string, data []byte) error
}
