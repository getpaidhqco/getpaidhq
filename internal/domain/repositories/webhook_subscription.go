package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type WebhookSubscriptionRepository interface {
	Create(ctx context.Context, subscription entities.WebhookSubscription) (entities.WebhookSubscription, error)
	GetByID(ctx context.Context, orgId string, id string) (entities.WebhookSubscription, error)
	FindByEvent(ctx context.Context, orgId string, event string) ([]entities.WebhookSubscription, error)
	Update(ctx context.Context, subscription entities.WebhookSubscription) (entities.WebhookSubscription, error)
	Delete(ctx context.Context, id string) error
}
