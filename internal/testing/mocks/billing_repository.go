package mocks

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

// MockUsageRecordRepository provides a reusable mock for usage record repository
type MockUsageRecordRepository struct {
	usageSummaries map[string]map[string]interface{}
}

// MockSubscriptionItemRepository provides a reusable mock for subscription item repository
type MockSubscriptionItemRepository struct{}

// NewMockSubscriptionItemRepository creates a new mock subscription item repository
func NewMockSubscriptionItemRepository() *MockSubscriptionItemRepository {
	return &MockSubscriptionItemRepository{}
}

func (m *MockSubscriptionItemRepository) FindById(ctx context.Context, orgId string, id string) (entities.SubscriptionItem, error) {
	return entities.SubscriptionItem{}, nil
}

func (m *MockSubscriptionItemRepository) Create(ctx context.Context, entity entities.SubscriptionItem) (entities.SubscriptionItem, error) {
	return entity, nil
}

func (m *MockSubscriptionItemRepository) Update(ctx context.Context, entity entities.SubscriptionItem) (entities.SubscriptionItem, error) {
	return entity, nil
}

func (m *MockSubscriptionItemRepository) FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]entities.SubscriptionItem, error) {
	return []entities.SubscriptionItem{}, nil
}

func (m *MockSubscriptionItemRepository) Delete(ctx context.Context, orgId string, id string) error {
	return nil
}

func (m *MockSubscriptionItemRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.SubscriptionItem, int, error) {
	return []entities.SubscriptionItem{}, 0, nil
}
