package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type SubscriptionRepository interface {
	FindById(ctx context.Context, acctId string, id string) (entities.Subscription, error)
	Create(ctx context.Context, entity entities.Subscription) (entities.Subscription, error)
}
