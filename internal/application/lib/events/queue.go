package events

import "context"

type QueueMessage struct {
	Data interface{}      `json:"data"`
	Type QueueMessageType `json:"type"`
}

type QueueMessageType string

const (
	IncomingWebhook QueueMessageType = "incoming_webhook"
)

type QueueMessageHandler func(msg QueueMessage) error

type QueueClient interface {
	Start(handler QueueMessageHandler)
	SendMessage(ctx context.Context, data QueueMessage) error
}
