package mocks

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/lib/request"

	"github.com/stretchr/testify/mock"
)

// MockSubscriptionRepository provides a reusable mock for subscription repository
type MockSubscriptionRepository struct {
	mock.Mock
}

func (m *MockSubscriptionRepository) Create(ctx context.Context, subscription entities.Subscription) (entities.Subscription, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) Update(ctx context.Context, subscription entities.Subscription) (entities.Subscription, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionRepository) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]entities.Subscription), args.Int(1), args.Error(2)
}

func (m *MockSubscriptionRepository) CreatePlanChange(ctx context.Context, planChange entities.SubscriptionPlanChange) (entities.SubscriptionPlanChange, error) {
	args := m.Called(ctx, planChange)
	return args.Get(0).(entities.SubscriptionPlanChange), args.Error(1)
}

func (m *MockSubscriptionRepository) FindPlanChangesBySubscription(ctx context.Context, orgId string, subscriptionId string) ([]entities.SubscriptionPlanChange, error) {
	args := m.Called(ctx, orgId, subscriptionId)
	return args.Get(0).([]entities.SubscriptionPlanChange), args.Error(1)
}

// NewMockSubscriptionRepository creates a new mock subscription repository
func NewMockSubscriptionRepository() *MockSubscriptionRepository {
	return &MockSubscriptionRepository{}
}

// MockCustomerRepository provides a reusable mock for customer repository
type MockCustomerRepository struct {
	mock.Mock
}

func (m *MockCustomerRepository) Create(ctx context.Context, customer entities.Customer) (entities.Customer, error) {
	args := m.Called(ctx, customer)
	return args.Get(0).(entities.Customer), args.Error(1)
}

func (m *MockCustomerRepository) FindById(ctx context.Context, orgId string, id string) (entities.Customer, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Customer), args.Error(1)
}

func (m *MockCustomerRepository) Update(ctx context.Context, customer entities.Customer) (entities.Customer, error) {
	args := m.Called(ctx, customer)
	return args.Get(0).(entities.Customer), args.Error(1)
}

func (m *MockCustomerRepository) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Customer, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]entities.Customer), args.Int(1), args.Error(2)
}

// NewMockCustomerRepository creates a new mock customer repository
func NewMockCustomerRepository() *MockCustomerRepository {
	return &MockCustomerRepository{}
}

// MockVariantRepository provides a reusable mock for variant repository
type MockVariantRepository struct {
	mock.Mock
}

func (m *MockVariantRepository) FindById(ctx context.Context, orgId string, id string) (entities.Variant, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Variant), args.Error(1)
}

func (m *MockVariantRepository) Create(ctx context.Context, variant entities.Variant) (entities.Variant, error) {
	args := m.Called(ctx, variant)
	return args.Get(0).(entities.Variant), args.Error(1)
}

func (m *MockVariantRepository) Update(ctx context.Context, variant entities.Variant) (entities.Variant, error) {
	args := m.Called(ctx, variant)
	return args.Get(0).(entities.Variant), args.Error(1)
}

// NewMockVariantRepository creates a new mock variant repository
func NewMockVariantRepository() *MockVariantRepository {
	return &MockVariantRepository{}
}

// MockPriceRepository provides a reusable mock for price repository
type MockPriceRepository struct {
	mock.Mock
}

func (m *MockPriceRepository) FindById(ctx context.Context, orgId string, id string) (entities.Price, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Price), args.Error(1)
}

func (m *MockPriceRepository) Create(ctx context.Context, price entities.Price) (entities.Price, error) {
	args := m.Called(ctx, price)
	return args.Get(0).(entities.Price), args.Error(1)
}

// NewMockPriceRepository creates a new mock price repository
func NewMockPriceRepository() *MockPriceRepository {
	return &MockPriceRepository{}
}