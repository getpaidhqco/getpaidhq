package nats

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
	"time"
)

type NatsNotificationPublisher struct {
	*nats.Conn
	logger logger.Logger
}

func NewNatsNotificationPublisher(logger logger.Logger) events.NotificationPublisher {
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
	return NatsNotificationPublisher{
		Conn:   nc,
		logger: logger,
	}
}

func (n NatsNotificationPublisher) Publish(orgId, topic string, message interface{}) error {
	data, _ := json.Marshal(events.Payload{
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

func (n NatsNotificationPublisher) Subscribe(topic string, handler func(topic string, data []byte)) (events.Subscription, error) {
	s, err := n.Conn.Subscribe(topic, func(m *nats.Msg) {
		handler(m.Subject, m.Data)
	})

	return s, err
}