package mocks

import (
	"context"
	"github.com/stretchr/testify/mock"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities/dunning"
)

// MockDunningService is a mock implementation of the DunningService interface
type MockDunningService struct {
	mock.Mock
}

// CreateCampaign mocks the CreateCampaign method
func (m *MockDunningService) CreateCampaign(ctx context.Context, input interfaces.CreateDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

// FindCampaignById mocks the FindCampaignById method
func (m *MockDunningService) FindCampaignById(ctx context.Context, orgId string, id string) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

// ListCampaigns mocks the ListCampaigns method
func (m *MockDunningService) ListCampaigns(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]dunning.DunningCampaign), args.Int(1), args.Error(2)
}

// ListCampaignsBySubscription mocks the ListCampaignsBySubscription method
func (m *MockDunningService) ListCampaignsBySubscription(ctx context.Context, orgId string, subscriptionId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	args := m.Called(ctx, orgId, subscriptionId, pagination)
	return args.Get(0).([]dunning.DunningCampaign), args.Int(1), args.Error(2)
}

// ListCampaignsByCustomer mocks the ListCampaignsByCustomer method
func (m *MockDunningService) ListCampaignsByCustomer(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	args := m.Called(ctx, orgId, customerId, pagination)
	return args.Get(0).([]dunning.DunningCampaign), args.Int(1), args.Error(2)
}

// PauseCampaign mocks the PauseCampaign method
func (m *MockDunningService) PauseCampaign(ctx context.Context, input interfaces.PauseDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

// ResumeCampaign mocks the ResumeCampaign method
func (m *MockDunningService) ResumeCampaign(ctx context.Context, input interfaces.ResumeDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

// CancelCampaign mocks the CancelCampaign method
func (m *MockDunningService) CancelCampaign(ctx context.Context, input interfaces.CancelDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

// UpdateCampaign mocks the UpdateCampaign method
func (m *MockDunningService) UpdateCampaign(ctx context.Context, orgId string, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, orgId, campaign)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

// ListAttemptsByCampaign mocks the ListAttemptsByCampaign method
func (m *MockDunningService) ListAttemptsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningAttempt, int, error) {
	args := m.Called(ctx, orgId, campaignId, pagination)
	return args.Get(0).([]dunning.DunningAttempt), args.Int(1), args.Error(2)
}

// TriggerManualAttempt mocks the TriggerManualAttempt method
func (m *MockDunningService) TriggerManualAttempt(ctx context.Context, input interfaces.TriggerManualAttemptInput) (dunning.DunningAttempt, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningAttempt), args.Error(1)
}

// ListCommunicationsByCampaign mocks the ListCommunicationsByCampaign method
func (m *MockDunningService) ListCommunicationsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningCommunication, int, error) {
	args := m.Called(ctx, orgId, campaignId, pagination)
	return args.Get(0).([]dunning.DunningCommunication), args.Int(1), args.Error(2)
}

// CreatePaymentUpdateToken mocks the CreatePaymentUpdateToken method
func (m *MockDunningService) CreatePaymentUpdateToken(ctx context.Context, input interfaces.CreatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

// VerifyPaymentUpdateToken mocks the VerifyPaymentUpdateToken method
func (m *MockDunningService) VerifyPaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, orgId, tokenId)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

// ActivatePaymentUpdateToken mocks the ActivatePaymentUpdateToken method
func (m *MockDunningService) ActivatePaymentUpdateToken(ctx context.Context, input interfaces.ActivatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

// RevokePaymentUpdateToken mocks the RevokePaymentUpdateToken method
func (m *MockDunningService) RevokePaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, orgId, tokenId)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

// CreateConfiguration mocks the CreateConfiguration method
func (m *MockDunningService) CreateConfiguration(ctx context.Context, input interfaces.CreateDunningConfigurationInput) (dunning.DunningConfiguration, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningConfiguration), args.Error(1)
}

// GetConfiguration mocks the GetConfiguration method
func (m *MockDunningService) GetConfiguration(ctx context.Context, orgId string, id string) (dunning.DunningConfiguration, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(dunning.DunningConfiguration), args.Error(1)
}

// ListConfigurations mocks the ListConfigurations method
func (m *MockDunningService) ListConfigurations(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningConfiguration, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]dunning.DunningConfiguration), args.Int(1), args.Error(2)
}

// UpdateConfiguration mocks the UpdateConfiguration method
func (m *MockDunningService) UpdateConfiguration(ctx context.Context, input interfaces.UpdateDunningConfigurationInput) (dunning.DunningConfiguration, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningConfiguration), args.Error(1)
}

// GetCustomerDunningHistory mocks the GetCustomerDunningHistory method
func (m *MockDunningService) GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (dunning.CustomerDunningHistory, error) {
	args := m.Called(ctx, orgId, customerId)
	return args.Get(0).(dunning.CustomerDunningHistory), args.Error(1)
}

// NewMockDunningService creates a new mock dunning service
func NewMockDunningService() *MockDunningService {
	return &MockDunningService{}
}