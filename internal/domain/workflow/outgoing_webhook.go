package workflow

import (
	pubsub "payloop/internal/application/lib/events"
	"payloop/internal/domain/entities"
)

type OutgoingWebhookPayload struct {
	WebhookSubscription entities.WebhookSubscription
	Event               pubsub.Payload
}
