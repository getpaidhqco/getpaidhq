package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type SubscriptionRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error)
	Create(ctx context.Context, entity entities.Subscription) (entities.Subscription, error)
	Update(ctx context.Context, entity entities.Subscription) (entities.Subscription, error)
	FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error)
}
