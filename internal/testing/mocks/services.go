package mocks

import (
	"context"
	"github.com/stretchr/testify/mock"
	"payloop/internal/domain/entities"
)

// MockDiscountService provides a reusable mock for discount service
type MockDiscountService struct{}

// NewMockDiscountService creates a new mock discount service
func NewMockDiscountService() *MockDiscountService {
	return &MockDiscountService{}
}

func (m *MockDiscountService) CalculateDiscount(ctx context.Context, orgId string, subscription entities.Subscription, amount int64) (int64, error) {
	return 0, nil
}

// MockSettingsService provides a reusable mock for settings service
type MockSettingsService struct {
	mock.Mock
}

func NewMockSettingsService() *MockSettingsService {
	return &MockSettingsService{}
}

func (m *MockSettingsService) GetSetting(ctx context.Context, orgId string, parentId string, id string, result interface{}) error {
	args := m.Called(ctx, orgId, parentId, id, result)
	return args.Error(0)
}

func (m *MockSettingsService) GetSettingRaw(ctx context.Context, orgId string, parentId string, id string) (interface{}, error) {
	args := m.Called(ctx, orgId, parentId, id)
	return args.Get(0), args.Error(1)
}

func (m *MockSettingsService) ListSettings(ctx context.Context, orgId string, parentId string) ([]entities.Setting, error) {
	args := m.Called(ctx, orgId, parentId)
	return args.Get(0).([]entities.Setting), args.Error(1)
}

func (m *MockSettingsService) CreateSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error) {
	args := m.Called(ctx, orgId, parentId, id, value)
	return args.Get(0).(entities.Setting), args.Error(1)
}

func (m *MockSettingsService) UpdateSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error) {
	args := m.Called(ctx, orgId, parentId, id, value)
	return args.Get(0).(entities.Setting), args.Error(1)
}

func (m *MockSettingsService) UpsertSetting(ctx context.Context, orgId string, parentId string, id string, value interface{}) (entities.Setting, error) {
	args := m.Called(ctx, orgId, parentId, id, value)
	return args.Get(0).(entities.Setting), args.Error(1)
}

func (m *MockSettingsService) DeleteSetting(ctx context.Context, orgId string, parentId string, id string) error {
	args := m.Called(ctx, orgId, parentId, id)
	return args.Error(0)
}