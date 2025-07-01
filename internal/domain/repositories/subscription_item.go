package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

// SubscriptionItemRepository defines the interface for subscription item repository operations
type SubscriptionItemRepository interface {
	// FindById finds a subscription item by ID
	FindById(ctx context.Context, orgId string, id string) (entities.SubscriptionItem, error)
	
	// Create creates a new subscription item
	Create(ctx context.Context, entity entities.SubscriptionItem) (entities.SubscriptionItem, error)
	
	// Update updates an existing subscription item
	Update(ctx context.Context, entity entities.SubscriptionItem) (entities.SubscriptionItem, error)
	
	// FindBySubscriptionId finds all subscription items for a subscription
	FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]entities.SubscriptionItem, error)
	
	// Find finds subscription items with pagination
	Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.SubscriptionItem, int, error)
	
	// Delete deletes a subscription item
	Delete(ctx context.Context, orgId string, id string) error
}