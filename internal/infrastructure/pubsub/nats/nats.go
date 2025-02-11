package nats

import (
	"encoding/json"
	"fmt"
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

func (n NatsPubSub) Publish(orgId, topic string, message interface{}) error {
	data, _ := json.Marshal(pubsub.Payload{
		Id:    lib.GenerateId("evt"),
		OrgId: orgId,
		Topic: topic,
		Data:  message,
	})
	n.logger.Debug(fmt.Sprintf("[nats] publishing topic [%s]", topic))
	err := n.Conn.Publish(topic, data)
	return err
}

func (n NatsPubSub) Subscribe(topic string, handler func(topic string, data []byte)) (pubsub.Subscription, error) {
	s, err := n.Conn.Subscribe(topic, func(m *nats.Msg) {
		handler(m.Subject, m.Data)
	})

	return s, err
}
