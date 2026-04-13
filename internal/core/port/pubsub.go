package port

import (
	"context"
	"time"
)

// PubSub defines the interface for publish/subscribe messaging.
type PubSub interface {
	Publish(orgId string, topic string, message interface{}) error
	Subscribe(topic string, handler func(topic string, data []byte)) (PubSubSubscription, error)
}

type PubSubPayload struct {
	Id        string      `json:"id"`
	OrgId     string      `json:"org_id"`
	Topic     string      `json:"topic"`
	Data      interface{} `json:"data"`
	CreatedAt time.Time   `json:"created_at"`
}

type PubSubSubscription interface {
	Unsubscribe() error
}

// QueueClient defines the interface for message queue operations.
type QueueClient interface {
	Start(handler QueueMessageHandler)
	SendMessage(ctx context.Context, data QueueMessage) error
}

type QueueMessage struct {
	Data interface{}      `json:"data"`
	Type QueueMessageType `json:"type"`
}

type QueueMessageType string

const (
	QueueIncomingWebhook     QueueMessageType = "incoming_webhook"
	QueueReportingDataChange QueueMessageType = "reporting_data_change"
)

type QueueMessageHandler func(msg QueueMessage) error
