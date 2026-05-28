package port

import (
	"getpaidhq/internal/core/domain"
)

// PubSub defines the interface for publish/subscribe messaging.
type PubSub interface {
	Publish(orgId string, topic string, message any) error
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
