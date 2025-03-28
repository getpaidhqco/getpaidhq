package interfaces

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/webhooks"
	"payloop/internal/domain/workflow"
)

type WebhookSubscriptionService interface {
	Create(ctx context.Context, input webhooks.CreateWebhookSubscriptionInput) (entities.WebhookSubscription, error)
	SendWebhook(ctx context.Context, input workflow.OutgoingWebhookPayload) error
}
