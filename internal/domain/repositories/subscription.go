package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type SubscriptionRepository interface {
	// Always loads with items - similar to Invoice/LineItems pattern
	FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error)

	// Create also creates any items specified in the subscription
	Create(ctx context.Context, entity entities.Subscription) (entities.Subscription, error)

	// Update only updates the subscription itself, not items
	Update(ctx context.Context, entity entities.Subscription) (entities.Subscription, error)

	FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error)
	Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.Subscription, int, error)

	// Plan change methods
	CreatePlanChange(ctx context.Context, entity entities.SubscriptionPlanChange) (entities.SubscriptionPlanChange, error)
	FindPlanChangesBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]entities.SubscriptionPlanChange, error)

	// Subscription item methods
	// Only if there's a specific performance need for subscription-only queries
	FindByIdWithoutItems(ctx context.Context, orgId string, id string) (entities.Subscription, error)
}
