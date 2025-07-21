package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/settings"
	"payloop/internal/infrastructure/workflow/temporal/testutils"
	"payloop/internal/lib"
)

// Mock interfaces for testing
type MockSubscriptionService struct {
	mock.Mock
}

func (m *MockSubscriptionService) ProcessSubscriptionCharge(ctx context.Context, subscription entities.Subscription) (payments.ChargeResult, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(payments.ChargeResult), args.Error(1)
}

func (m *MockSubscriptionService) HandleChargeResult(ctx context.Context, subscription entities.Subscription, result payments.ChargeResult) (entities.Subscription, error) {
	args := m.Called(ctx, subscription, result)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

type MockSettingsService struct {
	mock.Mock
}

func (m *MockSettingsService) GetSubscriptionSettings(ctx context.Context, orgId string) (settings.Subscription, error) {
	args := m.Called(ctx, orgId)
	return args.Get(0).(settings.Subscription), args.Error(1)
}

type MockNotificationPublisher struct {
	mock.Mock
}

func (m *MockNotificationPublisher) Publish(orgId, topic string, data interface{}) error {
	args := m.Called(orgId, topic, data)
	return args.Error(0)
}

type MockErrorReporter struct {
	mock.Mock
}

func (m *MockErrorReporter) ReportError(ctx context.Context, err error, metadata map[string]interface{}) {
	m.Called(ctx, err, metadata)
}

type OrderActivitiesTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env                    *testsuite.TestActivityEnvironment
	activities             *OrderActivities
	mockSubscriptionService *MockSubscriptionService
	mockSettingsService    *MockSettingsService
	mockNotificationPublisher *MockNotificationPublisher
	mockErrorReporter      *MockErrorReporter
}

func TestOrderActivitiesTestSuite(t *testing.T) {
	suite.Run(t, new(OrderActivitiesTestSuite))
}

func (s *OrderActivitiesTestSuite) SetupTest() {
	s.env = s.NewTestActivityEnvironment()
	
	// Create mocks
	s.mockSubscriptionService = new(MockSubscriptionService)
	s.mockSettingsService = new(MockSettingsService)
	s.mockNotificationPublisher = new(MockNotificationPublisher)
	s.mockErrorReporter = new(MockErrorReporter)
	
	// Create activities instance with mocks
	s.activities = &OrderActivities{
		subscriptionService: s.mockSubscriptionService,
		settingsService:     s.mockSettingsService,
		pubsub:             s.mockNotificationPublisher,
		errorReporter:      s.mockErrorReporter,
	}
	
	// Register activities
	s.env.RegisterActivity(s.activities.ChargeCustomerForBillingPeriod)
	s.env.RegisterActivity(s.activities.HandleChargeResult)
	s.env.RegisterActivity(s.activities.GetSubscriptionSettings)
	s.env.RegisterActivity(s.activities.NotifyWorkflowEnded)
}

func (s *OrderActivitiesTestSuite) TearDownTest() {
	s.env.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestChargeCustomerForBillingPeriod_Success() {
	// Test successful charge
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	expectedResult := testutils.MockSuccessfulChargeResult(1000)
	
	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, subscription).Return(
		expectedResult, nil)
	
	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.ChargeCustomerForBillingPeriod, subscription)
	
	// Assert results
	s.NoError(err)
	
	var chargeResult payments.ChargeResult
	s.NoError(result.Get(&chargeResult))
	s.Equal(expectedResult.Status, chargeResult.Status)
	s.Equal(expectedResult.Amount, chargeResult.Amount)
	
	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestChargeCustomerForBillingPeriod_GatewayError() {
	// Test gateway error handling
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	failedResult := testutils.MockFailedChargeResult(1000)
	
	// Create a gateway error
	gatewayErr := &lib.CustomError{
		Type:    lib.GatewayError,
		Message: "Gateway timeout",
	}
	
	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, subscription).Return(
		failedResult, gatewayErr)
	
	s.mockErrorReporter.On("ReportError", mock.Anything, mock.Anything, mock.Anything).Return()
	
	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.ChargeCustomerForBillingPeriod, subscription)
	
	// Assert that activity returns error for gateway failures (to trigger Temporal retry)
	s.Error(err)
	s.Contains(err.Error(), "gateway_error")
	
	s.mockSubscriptionService.AssertExpectations(s.T())
	s.mockErrorReporter.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestChargeCustomerForBillingPeriod_BusinessError() {
	// Test business error (non-gateway) handling
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	failedResult := testutils.MockFailedChargeResult(1000)
	
	// Create a non-gateway error
	businessErr := errors.New("insufficient funds")
	
	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, subscription).Return(
		failedResult, businessErr)
	
	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.ChargeCustomerForBillingPeriod, subscription)
	
	// Assert that business errors are returned with the charge result (not retried)
	s.NoError(err)
	
	var chargeResult payments.ChargeResult
	s.NoError(result.Get(&chargeResult))
	s.Equal(payments.PaymentStatusFailed, chargeResult.Status)
	
	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestHandleChargeResult_Success() {
	// Test successful charge result handling
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	chargeResult := testutils.MockSuccessfulChargeResult(1000)
	updatedSubscription := testutils.CreateUpdatedSubscription(subscription, entities.SubscriptionStatusActive)
	
	s.mockSubscriptionService.On("HandleChargeResult", mock.Anything, subscription, chargeResult).Return(
		updatedSubscription, nil)
	
	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.HandleChargeResult, subscription, chargeResult)
	
	// Assert results
	s.NoError(err)
	
	var resultSub entities.Subscription
	s.NoError(result.Get(&resultSub))
	s.Equal(updatedSubscription.Id, resultSub.Id)
	s.Equal(updatedSubscription.Status, resultSub.Status)
	s.Equal(updatedSubscription.CyclesProcessed, resultSub.CyclesProcessed)
	
	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestHandleChargeResult_Error() {
	// Test error in handling charge result
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	chargeResult := testutils.MockSuccessfulChargeResult(1000)
	
	s.mockSubscriptionService.On("HandleChargeResult", mock.Anything, subscription, chargeResult).Return(
		entities.Subscription{}, errors.New("database error"))
	
	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.HandleChargeResult, subscription, chargeResult)
	
	// Assert error
	s.Error(err)
	s.Contains(err.Error(), "database error")
	
	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestGetSubscriptionSettings_Success() {
	// Test successful settings retrieval
	orgId := "org_123"
	expectedSettings := testutils.MockSubscriptionSettings()
	
	s.mockSettingsService.On("GetSubscriptionSettings", mock.Anything, orgId).Return(
		expectedSettings, nil)
	
	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.GetSubscriptionSettings, orgId)
	
	// Assert results
	s.NoError(err)
	
	var settings settings.Subscription
	s.NoError(result.Get(&settings))
	s.Equal(expectedSettings.ReminderDays, settings.ReminderDays)
	
	s.mockSettingsService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestGetSubscriptionSettings_Error() {
	// Test error in settings retrieval
	orgId := "org_123"
	
	s.mockSettingsService.On("GetSubscriptionSettings", mock.Anything, orgId).Return(
		settings.Subscription{}, errors.New("settings not found"))
	
	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.GetSubscriptionSettings, orgId)
	
	// Assert error
	s.Error(err)
	s.Contains(err.Error(), "settings not found")
	
	s.mockSettingsService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestNotifyWorkflowEnded_Success() {
	// Test successful workflow ended notification
	orgId := "org_123"
	subscriptionId := "sub_456"
	
	s.mockNotificationPublisher.On("Publish", orgId, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	
	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.NotifyWorkflowEnded, orgId, subscriptionId)
	
	// Assert results
	s.NoError(err)
	s.Nil(result.Get(nil))
	
	s.mockNotificationPublisher.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestNotifyWorkflowEnded_Error() {
	// Test error in workflow ended notification
	orgId := "org_123"
	subscriptionId := "sub_456"
	
	s.mockNotificationPublisher.On("Publish", orgId, mock.AnythingOfType("string"), mock.Anything).Return(
		errors.New("notification service unavailable"))
	
	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.NotifyWorkflowEnded, orgId, subscriptionId)
	
	// Assert error
	s.Error(err)
	s.Contains(err.Error(), "notification service unavailable")
	
	s.mockNotificationPublisher.AssertExpectations(s.T())
}

// Additional integration-style tests
func (s *OrderActivitiesTestSuite) TestChargeCustomerFlow_FullScenario() {
	// Test full charge and handle result flow
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	chargeResult := testutils.MockSuccessfulChargeResult(1000)
	updatedSubscription := testutils.CreateUpdatedSubscription(subscription, entities.SubscriptionStatusActive)
	
	// Setup mocks for both activities
	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, subscription).Return(
		chargeResult, nil)
	s.mockSubscriptionService.On("HandleChargeResult", mock.Anything, subscription, chargeResult).Return(
		updatedSubscription, nil)
	
	// Execute charge activity
	chargeRes, err := s.env.ExecuteActivity(s.activities.ChargeCustomerForBillingPeriod, subscription)
	s.NoError(err)
	
	var actualChargeResult payments.ChargeResult
	s.NoError(chargeRes.Get(&actualChargeResult))
	
	// Execute handle result activity
	handleRes, err := s.env.ExecuteActivity(s.activities.HandleChargeResult, subscription, actualChargeResult)
	s.NoError(err)
	
	var actualSubscription entities.Subscription
	s.NoError(handleRes.Get(&actualSubscription))
	
	// Assert flow completed successfully
	s.Equal(payments.PaymentStatusSuccess, actualChargeResult.Status)
	s.Equal(entities.SubscriptionStatusActive, actualSubscription.Status)
	s.Equal(1, actualSubscription.CyclesProcessed)
	
	s.mockSubscriptionService.AssertExpectations(s.T())
}