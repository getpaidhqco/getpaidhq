package nats

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"time"
)

type NatsPubSub struct {
	*nats.Conn
	server *server.Server
	logger port.Logger
}

func NewNatsPubSub(logger port.Logger) port.PubSub {
	opts := &server.Options{
		Host: "localhost",
		Port: 4222,
	}
	// Initialize new server with options
	ns, err := server.NewServer(opts)

	if err != nil {
		panic(err)
	}
	go ns.Start()

	// Wait for server to be ready for connections
	if !ns.ReadyForConnections(15 * time.Second) {
		panic("not ready for connection")
	}

	nc, err := nats.Connect(ns.ClientURL())
	if err != nil {
		panic(err)
	}
	return NatsPubSub{
		Conn:   nc,
		server: ns,
		logger: logger,
	}
}

// Close drains the client connection (flushing pending publishes and
// unsubscribing all active subscriptions) and then shuts the embedded server
// down, waiting for it to stop.
func (n NatsPubSub) Close() error {
	if n.Conn != nil {
		// Drain closes the conn once in-flight messages have been processed.
		if err := n.Conn.Drain(); err != nil {
			n.logger.Warn(fmt.Sprintf("[nats] drain failed: %v", err))
		}
	}
	if n.server != nil {
		n.server.Shutdown()
		n.server.WaitForShutdown()
	}
	return nil
}

func (n NatsPubSub) Publish(orgId, topic string, message any) error {
	data, _ := json.Marshal(port.PubSubPayload{
		Id:        lib.GenerateId("evt"),
		OrgId:     orgId,
		Topic:     topic,
		Data:      message,
		CreatedAt: time.Now().UTC(),
	})
	n.logger.Debug(fmt.Sprintf("[nats] publishing topic [%s]", topic))
	err := n.Conn.Publish(topic, data)
	return err
}

func (n NatsPubSub) Subscribe(topic string, handler func(topic string, data []byte)) (port.PubSubSubscription, error) {
	s, err := n.Conn.Subscribe(topic, func(m *nats.Msg) {
		handler(m.Subject, m.Data)
	})

	return s, err
}
