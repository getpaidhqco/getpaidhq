package mocks

import (
	"context"
	"github.com/stretchr/testify/mock"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/settings"
	"payloop/internal/domain/entities/subscriptions"
)

// MockSubscriptionService is a mock implementation of the SubscriptionService interface
type MockSubscriptionService struct {
	mock.Mock
}

// CreateSubscriptionsForOrder mocks the CreateSubscriptionsForOrder method
func (m *MockSubscriptionService) CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
	args := m.Called(ctx, orgId, orderId)
	return args.Get(0).([]entities.Subscription), args.Error(1)
}

// FindById mocks the FindById method
func (m *MockSubscriptionService) FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

// Activate mocks the Activate method
func (m *MockSubscriptionService) Activate(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

// PauseSubscription mocks the PauseSubscription method
func (m *MockSubscriptionService) PauseSubscription(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

// List mocks the List method
func (m *MockSubscriptionService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]entities.Subscription), args.Int(1), args.Error(2)
}

// FindSubscriptionPayments mocks the FindSubscriptionPayments method
func (m *MockSubscriptionService) FindSubscriptionPayments(ctx context.Context, pk entities.EntityKey, pagination request.Pagination) ([]entities.Payment, int, error) {
	args := m.Called(ctx, pk, pagination)
	return args.Get(0).([]entities.Payment), args.Int(1), args.Error(2)
}

// ResumeSubscription mocks the ResumeSubscription method
func (m *MockSubscriptionService) ResumeSubscription(ctx context.Context, input subscriptions.ResumeSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

// CancelSubscription mocks the CancelSubscription method
func (m *MockSubscriptionService) CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

// ChangeSubscriptionPlan mocks the ChangeSubscriptionPlan method
func (m *MockSubscriptionService) ChangeSubscriptionPlan(ctx context.Context, input subscriptions.ChangePlanInput) (*entities.Subscription, *entities.SubscriptionPlanChange, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*entities.Subscription), args.Get(1).(*entities.SubscriptionPlanChange), args.Error(2)
}

// UpdateBillingAnchor mocks the UpdateBillingAnchor method
func (m *MockSubscriptionService) UpdateBillingAnchor(ctx context.Context, input dto.UpdateBillingAnchorInput) (dto.UpdateBillingAnchorResult, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dto.UpdateBillingAnchorResult), args.Error(1)
}

// GetSubscriptionCustomer mocks the GetSubscriptionCustomer method
func (m *MockSubscriptionService) GetSubscriptionCustomer(ctx context.Context, subscription entities.Subscription) (entities.Customer, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.Customer), args.Error(1)
}

// GetSubscriptionPaymentMethod mocks the GetSubscriptionPaymentMethod method
func (m *MockSubscriptionService) GetSubscriptionPaymentMethod(ctx context.Context, subscription entities.Subscription) (entities.SecurePaymentMethod, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.SecurePaymentMethod), args.Error(1)
}

// HandleSubscriptionChargeSuccess mocks the HandleSubscriptionChargeSuccess method
func (m *MockSubscriptionService) HandleSubscriptionChargeSuccess(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

// HandleSubscriptionChargeFailure mocks the HandleSubscriptionChargeFailure method
func (m *MockSubscriptionService) HandleSubscriptionChargeFailure(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

// GetOrgSubscriptionSettings mocks the GetOrgSubscriptionSettings method
func (m *MockSubscriptionService) GetOrgSubscriptionSettings(ctx context.Context, orgId string) (settings.Subscription, error) {
	args := m.Called(ctx, orgId)
	return args.Get(0).(settings.Subscription), args.Error(1)
}

// NewMockSubscriptionService creates a new mock subscription service
func NewMockSubscriptionService() *MockSubscriptionService {
	return &MockSubscriptionService{}
}
