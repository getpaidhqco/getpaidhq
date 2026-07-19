package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// NatsPubSub is a port.PubSub backed by an external NATS server.
type NatsPubSub struct {
	conn   *nats.Conn
	closed chan struct{}
	logger port.Logger
}

// NewNatsPubSub connects to the NATS server at url (falling back to the local
// default when empty) and returns a ready pub/sub. The connection retries the
// initial dial and reconnects indefinitely, so a transient broker outage
// neither crashes startup nor permanently drops the process.
func NewNatsPubSub(url string, logger port.Logger) (*NatsPubSub, error) {
	if url == "" {
		url = nats.DefaultURL // nats://127.0.0.1:4222
	}

	// Closed once the connection is fully torn down (after Drain finishes),
	// so Close can block until teardown actually completes.
	closed := make(chan struct{})

	nc, err := nats.Connect(url,
		nats.Name("getpaidhq"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				logger.Warn(fmt.Sprintf("[nats] disconnected: %v", err))
			}
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			logger.Info(fmt.Sprintf("[nats] reconnected to %s", c.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			close(closed)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to nats at %q: %w", url, err)
	}

	logger.Infof("[nats] connected to %s", url)
	return &NatsPubSub{conn: nc, closed: closed, logger: logger}, nil
}

// Conn exposes the underlying NATS connection so a JetStream adapter can share it
// (one connection, one reconnect policy, one drain). Returns nil if unset.
func (n *NatsPubSub) Conn() *nats.Conn { return n.conn }

func (n *NatsPubSub) Publish(_ context.Context, orgId, topic string, message any) error {
	data, err := json.Marshal(port.PubSubPayload{
		Id:        lib.GenerateId("evt"),
		OrgId:     orgId,
		Topic:     topic,
		Data:      message,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal pubsub payload for %q: %w", topic, err)
	}
	return n.PublishPayload(topic, data)
}

// PublishPayload implements port.RawPublisher.
func (n *NatsPubSub) PublishPayload(topic string, data []byte) error {
	n.logger.Debug(fmt.Sprintf("[nats] publishing topic [%s]", topic))
	return n.conn.Publish(topic, data)
}

func (n *NatsPubSub) Subscribe(topic string, handler func(topic string, data []byte)) (port.PubSubSubscription, error) {
	return n.conn.Subscribe(topic, func(m *nats.Msg) {
		handler(m.Subject, m.Data)
	})
}

// Close drains the connection (flushing pending publishes and unsubscribing
// active subscriptions) and waits for the underlying socket to close, so no
// goroutine outlives the call.
func (n *NatsPubSub) Close() error {
	if n.conn == nil {
		return nil
	}
	if err := n.conn.Drain(); err != nil {
		return fmt.Errorf("drain nats connection: %w", err)
	}
	select {
	case <-n.closed:
	case <-time.After(5 * time.Second):
		n.logger.Warn("[nats] drain did not complete within 5s; closing")
		n.conn.Close()
	}
	return nil
}
