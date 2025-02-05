package nats

import (
	"encoding/json"
	"github.com/nats-io/nats.go"
	pubsub "payloop/internal/application/lib/events"
	"payloop/internal/lib"
)

type NatsPubSub struct {
	*nats.Conn
	logger lib.Logger
}

func NewNatsPubSub(logger lib.Logger) pubsub.PubSub {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	return NatsPubSub{
		Conn:   nc,
		logger: logger,
	}
}

func (n NatsPubSub) Publish(topic string, message string) error {
	n.logger.Debug("Publishing to NATS", "topic", topic)
	err := n.Conn.Publish(topic, []byte(message))
	return err
}

func (n NatsPubSub) PublishJSON(topic string, message interface{}) error {
	data, _ := json.Marshal(message)
	n.logger.Debug("Publishing to NATS", "topic", topic, "data", string(data))
	err := n.Conn.Publish(topic, data)
	return err
}

func (n NatsPubSub) Subscribe(topic string, handler func(topic string, data []byte)) (pubsub.Subscription, error) {
	s, err := n.Conn.Subscribe(topic, func(m *nats.Msg) {
		handler(m.Subject, m.Data)
	})

	return s, err
}
