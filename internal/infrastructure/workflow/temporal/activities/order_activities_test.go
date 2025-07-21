package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/settings"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/domain/factories"
	"payloop/internal/infrastructure/workflow/temporal/testutils"
	"payloop/internal/lib"
)

// Mock interfaces for testing
type OrderMockSubscriptionService struct {
	mock.Mock
}

func (m *OrderMockSubscriptionService) ProcessSubscriptionCharge(ctx context.Context, subscription entities.Subscription) (payments.ChargeResult, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(payments.ChargeResult), args.Error(1)
}

// Add missing methods to implement interfaces.SubscriptionService
func (m *OrderMockSubscriptionService) CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
	args := m.Called(ctx, orgId, orderId)
	return args.Get(0).([]entities.Subscription), args.Error(1)
}

func (m *OrderMockSubscriptionService) Create(ctx context.Context, orgId string, input dto.CreateSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, orgId, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *OrderMockSubscriptionService) FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *OrderMockSubscriptionService) Activate(ctx context.Context, input subscriptions.ActivateSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *OrderMockSubscriptionService) PauseSubscription(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *OrderMockSubscriptionService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]entities.Subscription), args.Get(1).(int), args.Error(2)
}

func (m *OrderMockSubscriptionService) FindSubscriptionPayments(ctx context.Context, pk entities.EntityKey, pagination request.Pagination) ([]entities.Payment, int, error) {
	args := m.Called(ctx, pk, pagination)
	return args.Get(0).([]entities.Payment), args.Get(1).(int), args.Error(2)
}

func (m *OrderMockSubscriptionService) ResumeSubscription(ctx context.Context, input subscriptions.ResumeSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *OrderMockSubscriptionService) CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *OrderMockSubscriptionService) ChangeSubscriptionPlan(ctx context.Context, input subscriptions.ChangePlanInput) (*entities.Subscription, *entities.SubscriptionPlanChange, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*entities.Subscription), args.Get(1).(*entities.SubscriptionPlanChange), args.Error(2)
}

func (m *OrderMockSubscriptionService) UpdateBillingAnchor(ctx context.Context, input dto.UpdateBillingAnchorInput) (dto.UpdateBillingAnchorResult, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dto.UpdateBillingAnchorResult), args.Error(1)
}

func (m *OrderMockSubscriptionService) GetSubscriptionCustomer(ctx context.Context, subscription entities.Subscription) (entities.Customer, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.Customer), args.Error(1)
}

func (m *OrderMockSubscriptionService) GetSubscriptionPaymentMethod(ctx context.Context, subscription entities.Subscription) (entities.SecurePaymentMethod, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.SecurePaymentMethod), args.Error(1)
}

func (m *OrderMockSubscriptionService) HandleSubscriptionChargeSuccess(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *OrderMockSubscriptionService) HandleSubscriptionChargeFailure(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *OrderMockSubscriptionService) GetOrgSubscriptionSettings(ctx context.Context, orgId string) (settings.Subscription, error) {
	args := m.Called(ctx, orgId)
	return args.Get(0).(settings.Subscription), args.Error(1)
}

type OrderMockSettingsService struct {
	mock.Mock
}

func (m *OrderMockSettingsService) GetSubscriptionSettings(ctx context.Context, orgId string) (settings.Subscription, error) {
	args := m.Called(ctx, orgId)
	return args.Get(0).(settings.Subscription), args.Error(1)
}

// SettingRepository interface methods
func (m *OrderMockSettingsService) FindById(ctx context.Context, orgId string, parentId string, id string) (entities.Setting, error) {
	args := m.Called(ctx, orgId, parentId, id)
	return args.Get(0).(entities.Setting), args.Error(1)
}

func (m *OrderMockSettingsService) Create(ctx context.Context, entity entities.Setting) (entities.Setting, error) {
	args := m.Called(ctx, entity)
	return args.Get(0).(entities.Setting), args.Error(1)
}

func (m *OrderMockSettingsService) Update(ctx context.Context, entity entities.Setting) (entities.Setting, error) {
	args := m.Called(ctx, entity)
	return args.Get(0).(entities.Setting), args.Error(1)
}

func (m *OrderMockSettingsService) Delete(ctx context.Context, orgId string, parentId string, id string) error {
	args := m.Called(ctx, orgId, parentId, id)
	return args.Error(0)
}

func (m *OrderMockSettingsService) FindAll(ctx context.Context, orgId string, parentId string) ([]entities.Setting, error) {
	args := m.Called(ctx, orgId, parentId)
	return args.Get(0).([]entities.Setting), args.Error(1)
}

type OrderMockNotificationPublisher struct {
	mock.Mock
}

func (m *OrderMockNotificationPublisher) Publish(orgId, topic string, data interface{}) error {
	args := m.Called(orgId, topic, data)
	return args.Error(0)
}

func (m *OrderMockNotificationPublisher) Subscribe(topic string, handler func(topic string, data []byte)) (events.Subscription, error) {
	args := m.Called(topic, handler)
	return args.Get(0).(events.Subscription), args.Error(1)
}

type OrderMockErrorReporter struct {
	mock.Mock
}

func (m *OrderMockErrorReporter) ReportError(ctx interface{}, err error, metadata map[string]interface{}) {
	m.Called(ctx, err, metadata)
}

type OrderMockGatewayFactory struct {
	mock.Mock
}

// We don't need to implement any methods for GatewayFactory in these tests
// Just need it to satisfy the type requirement

type OrderMockLogger struct{}

func (l *OrderMockLogger) Debug(msg string, args ...any)               {}
func (l *OrderMockLogger) Info(msg string, args ...any)                {}
func (l *OrderMockLogger) Warn(msg string, args ...any)                {}
func (l *OrderMockLogger) Error(msg string, args ...any)               {}
func (l *OrderMockLogger) Fatal(msg string, args ...any)               {}
func (l *OrderMockLogger) Debugf(template string, args ...interface{}) {}
func (l *OrderMockLogger) Infof(template string, args ...interface{})  {}
func (l *OrderMockLogger) Warnf(template string, args ...interface{})  {}
func (l *OrderMockLogger) Errorf(template string, args ...interface{}) {}
func (l *OrderMockLogger) Panicf(template string, args ...interface{}) {}
func (l *OrderMockLogger) Fatalf(template string, args ...interface{}) {}
func (l *OrderMockLogger) Sync() error                                 { return nil }

var _ logger.Logger = (*OrderMockLogger)(nil)

type OrderActivitiesTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env                       *testsuite.TestActivityEnvironment
	activities                *OrderActivities
	mockSubscriptionService   *OrderMockSubscriptionService
	mockSettingsService       *OrderMockSettingsService
	mockNotificationPublisher *OrderMockNotificationPublisher
	mockErrorReporter         *OrderMockErrorReporter
	mockGatewayFactory        *OrderMockGatewayFactory
}

func TestOrderActivitiesTestSuite(t *testing.T) {
	suite.Run(t, new(OrderActivitiesTestSuite))
}

func (s *OrderActivitiesTestSuite) SetupTest() {
	s.env = s.NewTestActivityEnvironment()

	// Create mocks
	s.mockSubscriptionService = new(OrderMockSubscriptionService)
	s.mockSettingsService = new(OrderMockSettingsService)
	s.mockNotificationPublisher = new(OrderMockNotificationPublisher)
	s.mockErrorReporter = new(OrderMockErrorReporter)
	s.mockGatewayFactory = new(OrderMockGatewayFactory)

	// Create activities instance with mocks
	activities := OrderActivities{
		orderService:           nil, // Not needed for these tests
		subscriptionService:    s.mockSubscriptionService,
		dunningService:         nil, // Not needed for these tests
		subscriptionRepository: nil, // Not needed for these tests
		settingRepository:      s.mockSettingsService,
		paymentRepository:      nil, // Not needed for these tests
		pubsub:                 s.mockNotificationPublisher,
		gatewayFactory:         factories.GatewayFactory{},               // Use empty struct
		errorReporter:          lib.NewErrorReporter(&OrderMockLogger{}), // Use mock logger
	}
	s.activities = &activities

	// Register activities
	s.env.RegisterActivity(s.activities.ChargeCustomerForBillingPeriod)
	s.env.RegisterActivity(s.activities.HandleChargeResult)
	s.env.RegisterActivity(s.activities.GetSubscriptionSettings)
	s.env.RegisterActivity(s.activities.NotifyWorkflowEnded)
}

func (s *OrderActivitiesTestSuite) TearDownTest() {
	// Verify all mocks were called as expected
	s.mockSubscriptionService.AssertExpectations(s.T())
	s.mockSettingsService.AssertExpectations(s.T())
	s.mockNotificationPublisher.AssertExpectations(s.T())
	s.mockErrorReporter.AssertExpectations(s.T())
	s.mockGatewayFactory.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestChargeCustomerForBillingPeriod_Success() {
	// Test successful charge
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	expectedResult := testutils.MockSuccessfulChargeResult(1000)

	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, mock.Anything).Return(
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

	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, mock.Anything).Return(
		failedResult, gatewayErr)

	// Don't mock ErrorReporter - it's complex and the actual implementation works fine

	// Execute activity
	_, err := s.env.ExecuteActivity(s.activities.ChargeCustomerForBillingPeriod, subscription)

	// Assert that activity returns error for gateway failures (to trigger Temporal retry)
	s.Error(err)
	s.Contains(err.Error(), "gateway_error")

	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestChargeCustomerForBillingPeriod_BusinessError() {
	// Test business error (non-gateway) handling
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	failedResult := testutils.MockFailedChargeResult(1000)

	// Create a non-gateway error
	businessErr := errors.New("insufficient funds")

	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, mock.Anything).Return(
		failedResult, businessErr)

	// Execute activity
	_, err := s.env.ExecuteActivity(s.activities.ChargeCustomerForBillingPeriod, subscription)

	// Assert that business errors are returned as errors (will trigger workflow error handling)
	s.Error(err)
	s.Contains(err.Error(), "insufficient funds")

	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestHandleChargeResult_Success() {
	// Test successful charge result handling
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	chargeResult := testutils.MockSuccessfulChargeResult(1000)
	updatedSubscription := testutils.CreateUpdatedSubscription(subscription, entities.SubscriptionStatusActive)

	s.mockSubscriptionService.On("HandleSubscriptionChargeSuccess", mock.Anything, mock.Anything).Return(
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

	s.mockSubscriptionService.On("HandleSubscriptionChargeSuccess", mock.Anything, mock.Anything).Return(
		entities.Subscription{}, errors.New("database error"))

	// Execute activity
	_, err := s.env.ExecuteActivity(s.activities.HandleChargeResult, subscription, chargeResult)

	// Assert error
	s.Error(err)
	s.Contains(err.Error(), "database error")

	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestGetSubscriptionSettings_Success() {
	// Test successful settingsSub retrieval
	orgId := "org_123"
	expectedSettings := testutils.MockSubscriptionSettings()

	s.mockSubscriptionService.On("GetOrgSubscriptionSettings", mock.Anything, orgId).Return(
		expectedSettings, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.GetSubscriptionSettings, orgId)

	// Assert results
	s.NoError(err)
	
	var settingsSub settings.Subscription
	s.NoError(result.Get(&settingsSub))
	s.Equal(expectedSettings.ReminderDays, settingsSub.ReminderDays)

	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestGetSubscriptionSettings_Error() {
	// Test error in settings retrieval
	orgId := "org_123"

	s.mockSubscriptionService.On("GetOrgSubscriptionSettings", mock.Anything, orgId).Return(
		settings.Subscription{}, errors.New("settings not found"))

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.GetSubscriptionSettings, orgId)

	// Assert no error (method handles errors internally and returns default)
	s.NoError(err)

	var settings settings.Subscription
	s.NoError(result.Get(&settings))

	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestNotifyWorkflowEnded_Success() {
	// Test successful workflow ended notification
	orgId := "org_123"
	subscriptionId := "sub_456"

	s.mockNotificationPublisher.On("Publish", orgId, mock.AnythingOfType("string"), mock.Anything).Return(nil)

	// Execute activity
	_, err := s.env.ExecuteActivity(s.activities.NotifyWorkflowEnded, orgId, subscriptionId)

	// Assert results
	s.NoError(err)
	// NotifyWorkflowEnded returns no meaningful value

	s.mockNotificationPublisher.AssertExpectations(s.T())
}

func (s *OrderActivitiesTestSuite) TestNotifyWorkflowEnded_Error() {
	// Test error in workflow ended notification (method ignores pubsub errors)
	orgId := "org_123"
	subscriptionId := "sub_456"

	s.mockNotificationPublisher.On("Publish", orgId, mock.AnythingOfType("string"), mock.Anything).Return(
		errors.New("notification service unavailable"))

	// Execute activity
	_, err := s.env.ExecuteActivity(s.activities.NotifyWorkflowEnded, orgId, subscriptionId)

	// Assert no error (method ignores pubsub errors and always returns nil)
	s.NoError(err)
	// NotifyWorkflowEnded returns no meaningful value

	s.mockNotificationPublisher.AssertExpectations(s.T())
}

// Additional integration-style tests
func (s *OrderActivitiesTestSuite) TestChargeCustomerFlow_FullScenario() {
	// Test full charge and handle result flow
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	chargeResult := testutils.MockSuccessfulChargeResult(1000)
	updatedSubscription := testutils.CreateUpdatedSubscription(subscription, entities.SubscriptionStatusActive)

	// Setup mocks for both activities
	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, mock.Anything).Return(
		chargeResult, nil)
	s.mockSubscriptionService.On("HandleSubscriptionChargeSuccess", mock.Anything, mock.Anything).Return(
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
	s.Equal(payments.PaymentStatusSucceeded, actualChargeResult.Status)
	s.Equal(entities.SubscriptionStatusActive, actualSubscription.Status)
	s.Equal(1, actualSubscription.CyclesProcessed)

	s.mockSubscriptionService.AssertExpectations(s.T())
}
