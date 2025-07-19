package interfaces

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/webhooks"
	"payloop/internal/domain/workflow"
)

type WebhookSubscriptionService interface {
	Create(ctx context.Context, input webhooks.CreateWebhookSubscriptionInput) (entities.WebhookSubscription, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.WebhookSubscription, int, error)
	GetByID(ctx context.Context, orgId string, id string) (entities.WebhookSubscription, error)
	Update(ctx context.Context, subscription entities.WebhookSubscription) (entities.WebhookSubscription, error)
	Delete(ctx context.Context, orgId string, id string) error
	SendWebhook(ctx context.Context, input workflow.OutgoingWebhookPayload) error
}
