package service

import (
	"runtime/debug"

	"getpaidhq/internal/core/port"
)

// safePubSubHandler wraps a pubsub-handler callback in a panic recover
// so that one buggy handler doesn't kill the NATS receive loop and
// silently silence every other subscriber on the same connection.
// Panics are logged with a stack trace; we DO NOT re-raise. The price
// of dropping a single event is much smaller than the price of
// turning the whole pubsub subsystem into a one-shot fuse.
//
// Use this at every Subscribe call in this package. Production code
// must never pass a bare closure to pubsub.Subscribe.
func safePubSubHandler(logger port.Logger, name string, h func(string, []byte)) func(string, []byte) {
	return func(topic string, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("pubsub handler panic recovered",
					"handler", name,
					"topic", topic,
					"recover", r,
					"stack", string(debug.Stack()))
			}
		}()
		h(topic, data)
	}
}
