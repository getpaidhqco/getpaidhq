package mocks

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/api/dto/request"
	"time"
)

// MockUsageRecordRepository provides a reusable mock for usage record repository
type MockUsageRecordRepository struct {
	usageSummaries map[string]map[string]interface{}
}

// NewMockUsageRecordRepository creates a new mock usage record repository
func NewMockUsageRecordRepository() *MockUsageRecordRepository {
	return &MockUsageRecordRepository{
		usageSummaries: make(map[string]map[string]interface{}),
	}
}

func (m *MockUsageRecordRepository) FindById(ctx context.Context, orgId string, id string) (entities.UsageRecord, error) {
	return entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) Create(ctx context.Context, entity entities.UsageRecord) (entities.UsageRecord, error) {
	return entity, nil
}

func (m *MockUsageRecordRepository) Update(ctx context.Context, entity entities.UsageRecord) (entities.UsageRecord, error) {
	return entity, nil
}

func (m *MockUsageRecordRepository) FindBySubscriptionItemId(ctx context.Context, orgId string, subscriptionItemId string) ([]entities.UsageRecord, error) {
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]entities.UsageRecord, error) {
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) FindByBillingPeriod(ctx context.Context, orgId string, subscriptionId string, billingPeriod string) ([]entities.UsageRecord, error) {
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) FindUnprocessed(ctx context.Context, orgId string, subscriptionId string, billingPeriod string) ([]entities.UsageRecord, error) {
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) MarkProcessed(ctx context.Context, orgId string, ids []string, invoiceId string) error {
	return nil
}

func (m *MockUsageRecordRepository) AggregateUsage(ctx context.Context, orgId string, subscriptionItemId string, billingPeriod string, aggregationType string) (float64, error) {
	return 0, nil
}

func (m *MockUsageRecordRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.UsageRecord, int, error) {
	return []entities.UsageRecord{}, 0, nil
}

func (m *MockUsageRecordRepository) Delete(ctx context.Context, orgId string, id string) error {
	return nil
}

func (m *MockUsageRecordRepository) BatchCreate(ctx context.Context, entities []entities.UsageRecord) ([]entities.UsageRecord, error) {
	return entities, nil
}

func (m *MockUsageRecordRepository) GetUsageSummary(ctx context.Context, orgId string, subscriptionItemId string, startDate time.Time, endDate time.Time) (map[string]interface{}, error) {
	key := orgId + ":" + subscriptionItemId
	if summary, exists := m.usageSummaries[key]; exists {
		return summary, nil
	}
	return map[string]interface{}{
		"quantity": float64(0),
	}, nil
}

func (m *MockUsageRecordRepository) FindBySubscriptionItem(ctx context.Context, orgId string, subscriptionItemId string, startDate time.Time, endDate time.Time) ([]entities.UsageRecord, error) {
	// Return empty slice for tests that don't explicitly set up usage records
	return []entities.UsageRecord{}, nil
}

func (m *MockUsageRecordRepository) SetUsageSummary(orgId string, subscriptionItemId string, summary map[string]interface{}) {
	key := orgId + ":" + subscriptionItemId
	m.usageSummaries[key] = summary
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
