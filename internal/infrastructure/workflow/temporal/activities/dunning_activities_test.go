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
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/infrastructure/workflow/temporal/testutils"
	"payloop/internal/lib"
)

// Mock interfaces for dunning activities testing
type MockDunningService struct {
	mock.Mock
}

func (m *MockDunningService) CreateCampaign(ctx context.Context, input dto.CreateDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *MockDunningService) FindCampaignById(ctx context.Context, orgId, campaignId string) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, orgId, campaignId)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *MockDunningService) ListConfigurations(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningConfiguration, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]dunning.DunningConfiguration), args.Get(1).(int), args.Error(2)
}

// Add missing DunningService methods
func (m *MockDunningService) ListCampaigns(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]dunning.DunningCampaign), args.Get(1).(int), args.Error(2)
}

func (m *MockDunningService) ListCampaignsBySubscription(ctx context.Context, orgId string, subscriptionId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	args := m.Called(ctx, orgId, subscriptionId, pagination)
	return args.Get(0).([]dunning.DunningCampaign), args.Get(1).(int), args.Error(2)
}

func (m *MockDunningService) ListCampaignsByCustomer(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	args := m.Called(ctx, orgId, customerId, pagination)
	return args.Get(0).([]dunning.DunningCampaign), args.Get(1).(int), args.Error(2)
}

func (m *MockDunningService) UpdateCampaign(ctx context.Context, orgId string, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, orgId, campaign)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *MockDunningService) ListAttemptsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningAttempt, int, error) {
	args := m.Called(ctx, orgId, campaignId, pagination)
	return args.Get(0).([]dunning.DunningAttempt), args.Get(1).(int), args.Error(2)
}

func (m *MockDunningService) TriggerChargeAttempt(ctx context.Context, input dto.TriggerAttemptInput) (dunning.DunningAttempt, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningAttempt), args.Error(1)
}

func (m *MockDunningService) ListCommunicationsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningCommunication, int, error) {
	args := m.Called(ctx, orgId, campaignId, pagination)
	return args.Get(0).([]dunning.DunningCommunication), args.Get(1).(int), args.Error(2)
}

func (m *MockDunningService) CreatePaymentUpdateToken(ctx context.Context, input dto.CreatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

func (m *MockDunningService) VerifyPaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, orgId, tokenId)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

func (m *MockDunningService) ActivatePaymentUpdateToken(ctx context.Context, input dto.ActivatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

func (m *MockDunningService) RevokePaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, orgId, tokenId)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

func (m *MockDunningService) CreateConfiguration(ctx context.Context, input dto.CreateDunningConfigurationInput) (dunning.DunningConfiguration, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningConfiguration), args.Error(1)
}

func (m *MockDunningService) GetConfiguration(ctx context.Context, orgId string, id string) (dunning.DunningConfiguration, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(dunning.DunningConfiguration), args.Error(1)
}

func (m *MockDunningService) UpdateConfiguration(ctx context.Context, input dto.UpdateDunningConfigurationInput) (dunning.DunningConfiguration, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningConfiguration), args.Error(1)
}

func (m *MockDunningService) GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (dunning.CustomerDunningHistory, error) {
	args := m.Called(ctx, orgId, customerId)
	return args.Get(0).(dunning.CustomerDunningHistory), args.Error(1)
}

func (m *MockDunningService) PauseCampaign(ctx context.Context, input dto.PauseDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *MockDunningService) ResumeCampaign(ctx context.Context, input dto.ResumeDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *MockDunningService) CancelCampaign(ctx context.Context, input dto.CancelDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *MockDunningService) HandleChargeResult(ctx context.Context, campaign dunning.DunningCampaign, result payments.ChargeResult, config dunning.DunningConfig) (dto.HandleChargeResultResponse, error) {
	args := m.Called(ctx, campaign, result, config)
	return args.Get(0).(dto.HandleChargeResultResponse), args.Error(1)
}

type MockSubscriptionService struct {
	mock.Mock
}

func (m *MockSubscriptionService) CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
	args := m.Called(ctx, orgId, orderId)
	return args.Get(0).([]entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) Create(ctx context.Context, orgId string, input dto.CreateSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, orgId, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) Activate(ctx context.Context, input dto.ActivateSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) PauseSubscription(ctx context.Context, input dto.PauseSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]entities.Subscription), args.Get(1).(int), args.Error(2)
}

func (m *MockSubscriptionService) FindSubscriptionPayments(ctx context.Context, pk entities.EntityKey, pagination request.Pagination) ([]entities.Payment, int, error) {
	args := m.Called(ctx, pk, pagination)
	return args.Get(0).([]entities.Payment), args.Get(1).(int), args.Error(2)
}

func (m *MockSubscriptionService) ResumeSubscription(ctx context.Context, input dto.ResumeSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) CancelSubscription(ctx context.Context, input dto.CancelSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) ChangeSubscriptionPlan(ctx context.Context, input dto.ChangePlanInput) (*entities.Subscription, *entities.SubscriptionPlanChange, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*entities.Subscription), args.Get(1).(*entities.SubscriptionPlanChange), args.Error(2)
}

func (m *MockSubscriptionService) UpdateBillingAnchor(ctx context.Context, input dto.UpdateBillingAnchorInput) (dto.UpdateBillingAnchorResult, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dto.UpdateBillingAnchorResult), args.Error(1)
}

func (m *MockSubscriptionService) GetSubscriptionCustomer(ctx context.Context, subscription entities.Subscription) (entities.Customer, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.Customer), args.Error(1)
}

func (m *MockSubscriptionService) GetSubscriptionPaymentMethod(ctx context.Context, subscription entities.Subscription) (entities.SecurePaymentMethod, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.SecurePaymentMethod), args.Error(1)
}

func (m *MockSubscriptionService) HandleSubscriptionChargeSuccess(ctx context.Context, input dto.SubscriptionChargeInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) HandleSubscriptionChargeFailure(ctx context.Context, input dto.SubscriptionChargeInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *MockSubscriptionService) GetOrgSubscriptionSettings(ctx context.Context, orgId string) (entities.SubscriptionSettings, error) {
	args := m.Called(ctx, orgId)
	return args.Get(0).(entities.SubscriptionSettings), args.Error(1)
}

func (m *MockSubscriptionService) ProcessSubscriptionCharge(ctx context.Context, subscription entities.Subscription) (payments.ChargeResult, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(payments.ChargeResult), args.Error(1)
}

type MockNotificationPublisher struct {
	mock.Mock
}

func (m *MockNotificationPublisher) Publish(orgId string, topic string, message interface{}) error {
	args := m.Called(orgId, topic, message)
	return args.Error(0)
}

func (m *MockNotificationPublisher) Subscribe(topic string, handler func(topic string, data []byte)) (events.Subscription, error) {
	args := m.Called(topic, handler)
	return args.Get(0).(events.Subscription), args.Error(1)
}

type MockErrorReporter struct {
	mock.Mock
}

func (m *MockErrorReporter) ReportError(ctx interface{}, err error, data map[string]interface{}) {
	m.Called(ctx, err, data)
}

type MockTransactionService struct {
	mock.Mock
}

func (m *MockTransactionService) WithTransaction(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	args := m.Called(ctx, fn)
	if args.Get(1) != nil {
		return args.Get(0), args.Error(1)
	}
	// Execute the function if no error is expected
	return fn(ctx)
}

type DunningActivitiesTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env                       *testsuite.TestActivityEnvironment
	activities                *DunningActivities
	mockDunningService        *MockDunningService
	mockSubscriptionService   *MockSubscriptionService
	mockNotificationPublisher *MockNotificationPublisher
	mockErrorReporter         *MockErrorReporter
	mockTransactionService    *MockTransactionService
}

func TestDunningActivitiesTestSuite(t *testing.T) {
	suite.Run(t, new(DunningActivitiesTestSuite))
}

func (s *DunningActivitiesTestSuite) SetupTest() {
	s.env = s.NewTestActivityEnvironment()

	// Create mocks
	s.mockDunningService = new(MockDunningService)
	s.mockSubscriptionService = new(MockSubscriptionService)
	s.mockNotificationPublisher = new(MockNotificationPublisher)
	s.mockErrorReporter = new(MockErrorReporter)
	s.mockTransactionService = new(MockTransactionService)

	// Create activities instance with mocks
	s.activities = &DunningActivities{
		dunningService:      s.mockDunningService,
		subscriptionService: s.mockSubscriptionService,
		pubsub:              s.mockNotificationPublisher,
		errorReporter:       s.mockErrorReporter,
		TransactionService:  s.mockTransactionService,
	}

	// Register activities
	s.env.RegisterActivity(s.activities.CreateDunningCampaign)
	s.env.RegisterActivity(s.activities.ResolveDunningConfig)
	s.env.RegisterActivity(s.activities.PauseDunningCampaign)
	s.env.RegisterActivity(s.activities.ResumeDunningCampaign)
	s.env.RegisterActivity(s.activities.CancelDunningCampaign)
	s.env.RegisterActivity(s.activities.HandleSubscriptionStateChanged)
	s.env.RegisterActivity(s.activities.ProcessRetryCharge)
	s.env.RegisterActivity(s.activities.HandleDunningChargeResult)
	s.env.RegisterActivity(s.activities.SendDunningCommunication)
}

func (s *DunningActivitiesTestSuite) TearDownTest() {
	s.env.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestCreateDunningCampaign_Success() {
	// Test successful campaign creation
	input := testutils.CreateDunningWorkflowInput("org_123", "sub_456", "cust_789")
	expectedCampaign := testutils.MockDunningCampaign(input.OrgId, input.SubscriptionId, input.CustomerId)

	expectedInput := dto.CreateDunningCampaignInput{
		OrgId:                input.OrgId,
		SubscriptionId:       input.SubscriptionId,
		CustomerId:           input.CustomerId,
		FailedAmount:         input.FailedAmount,
		Currency:             input.Currency,
		InitialFailureReason: input.InitialFailureReason,
		ParentWorkflowId:     input.ParentWorkflowId,
		Metadata:             input.Metadata,
	}

	s.mockDunningService.On("CreateCampaign", mock.Anything, expectedInput).Return(
		expectedCampaign, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.CreateDunningCampaign, input)

	// Assert results
	s.NoError(err)

	var campaign dunning.DunningCampaign
	s.NoError(result.Get(&campaign))
	s.Equal(expectedCampaign.Id, campaign.Id)
	s.Equal(expectedCampaign.OrgId, campaign.OrgId)
	s.Equal(expectedCampaign.Status, campaign.Status)

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestCreateDunningCampaign_Error() {
	// Test campaign creation error
	input := testutils.CreateDunningWorkflowInput("org_123", "sub_456", "cust_789")

	s.mockDunningService.On("CreateCampaign", mock.Anything, mock.Anything).Return(
		dunning.DunningCampaign{}, errors.New("campaign creation failed"))

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.CreateDunningCampaign, input)

	// Assert error
	s.Error(err)
	s.Contains(err.Error(), "campaign creation failed")

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestResolveDunningConfig_WithActiveConfig() {
	// Test resolving dunning config with active configurations
	orgId := "org_123"

	configs := []dunning.DunningConfiguration{
		{
			OrgId:    orgId,
			Priority: 1,
			Status:   dunning.ConfigStatusActive,
		},
		{
			OrgId:    orgId,
			Priority: 2,
			Status:   dunning.ConfigStatusActive,
		},
	}

	s.mockDunningService.On("ListConfigurations", mock.Anything, orgId, mock.Anything).Return(
		configs, dto.PaginationMeta{}, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.ResolveDunningConfig, orgId)

	// Assert results - should return default config for now
	s.NoError(err)

	var config dunning.DunningConfig
	s.NoError(result.Get(&config))
	// Currently returns default config, but structure is in place for custom configs
	s.Equal(3, config.ProgressiveRetries.MaxAttempts)

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestResolveDunningConfig_NoConfigs() {
	// Test resolving dunning config with no configurations
	orgId := "org_123"

	s.mockDunningService.On("ListConfigurations", mock.Anything, orgId, mock.Anything).Return(
		[]dunning.DunningConfiguration{}, dto.PaginationMeta{}, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.ResolveDunningConfig, orgId)

	// Assert returns default config
	s.NoError(err)

	var config dunning.DunningConfig
	s.NoError(result.Get(&config))
	s.Equal(dunning.DefaultDunningConfig(), config)

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestPauseDunningCampaign_Success() {
	// Test successful campaign pause
	orgId := "org_123"
	campaignId := "campaign_456"

	expectedCampaign := testutils.MockDunningCampaign(orgId, "sub_123", "cust_456")
	expectedCampaign.Status = dunning.DunningStatusPaused

	expectedInput := dto.PauseDunningCampaignInput{
		OrgId:  orgId,
		Id:     campaignId,
		Reason: "manual_pause",
	}

	s.mockDunningService.On("PauseCampaign", mock.Anything, expectedInput).Return(
		expectedCampaign, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.PauseDunningCampaign, orgId, campaignId)

	// Assert results
	s.NoError(err)

	var campaign dunning.DunningCampaign
	s.NoError(result.Get(&campaign))
	s.Equal(dunning.DunningStatusPaused, campaign.Status)

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestProcessRetryCharge_Success() {
	// Test successful retry charge
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	expectedResult := testutils.MockSuccessfulChargeResult(1000)

	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, subscription).Return(
		expectedResult, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.ProcessRetryCharge, subscription)

	// Assert results
	s.NoError(err)

	var chargeResult payments.ChargeResult
	s.NoError(result.Get(&chargeResult))
	s.Equal(expectedResult.Status, chargeResult.Status)
	s.Equal(expectedResult.Amount, chargeResult.Amount)

	s.mockSubscriptionService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestProcessRetryCharge_GatewayError() {
	// Test gateway error in retry charge
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	failedResult := testutils.MockFailedChargeResult(1000)

	gatewayErr := &lib.CustomError{
		Type:    lib.GatewayError,
		Message: "Gateway timeout",
	}

	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, subscription).Return(
		failedResult, gatewayErr)

	s.mockErrorReporter.On("ReportError", mock.Anything, mock.Anything, mock.Anything).Return()

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.ProcessRetryCharge, subscription)

	// Assert that gateway errors are returned as Temporal application errors for retry
	s.Error(err)
	s.Contains(err.Error(), "gateway_error")

	s.mockSubscriptionService.AssertExpectations(s.T())
	s.mockErrorReporter.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestHandleDunningChargeResult_Success() {
	// Test successful charge result handling
	campaign := testutils.MockDunningCampaign("org_123", "sub_456", "cust_789")
	chargeResult := testutils.MockSuccessfulChargeResult(1000)
	config := testutils.MockDunningConfig()
	subscription := testutils.CreateFastSubscription("org_123", "cust_789", 1000)

	expectedResponse := dto.HandleDunningChargeResultResponse{
		Campaign:     campaign,
		Subscription: subscription,
		Attempt:      dunning.DunningAttempt{Id: "attempt_123"},
	}

	// Mock transaction execution
	s.mockTransactionService.On("WithTransaction", mock.Anything, mock.Anything).Return(
		HandleChargeAttemptResult{
			Campaign:     campaign,
			Subscription: subscription,
			Attempt:      expectedResponse.Attempt,
		}, nil)

	s.mockDunningService.On("HandleChargeResult", mock.Anything, campaign, chargeResult, config).Return(
		expectedResponse, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.HandleDunningChargeResult, campaign, chargeResult, config)

	// Assert results
	s.NoError(err)

	var handleResult HandleChargeAttemptResult
	s.NoError(result.Get(&handleResult))
	s.Equal(campaign.Id, handleResult.Campaign.Id)
	s.Equal(subscription.Id, handleResult.Subscription.Id)

	s.mockTransactionService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestHandleDunningChargeResult_TransactionError() {
	// Test transaction error in charge result handling
	campaign := testutils.MockDunningCampaign("org_123", "sub_456", "cust_789")
	chargeResult := testutils.MockSuccessfulChargeResult(1000)
	config := testutils.MockDunningConfig()

	s.mockTransactionService.On("WithTransaction", mock.Anything, mock.Anything).Return(
		nil, errors.New("transaction failed"))

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.HandleDunningChargeResult, campaign, chargeResult, config)

	// Assert error
	s.Error(err)
	s.Contains(err.Error(), "transaction failed")

	s.mockTransactionService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestSendDunningCommunication_Success() {
	// Test successful dunning communication
	orgId := "org_123"
	campaignId := "campaign_456"
	attemptNumber := 1

	campaign := testutils.MockDunningCampaign(orgId, "sub_456", "cust_789")

	s.mockDunningService.On("FindCampaignById", mock.Anything, orgId, campaignId).Return(
		campaign, nil)

	s.mockNotificationPublisher.On("Publish", orgId, mock.AnythingOfType("string"), mock.Anything).Return(nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.SendDunningCommunication, orgId, campaignId, attemptNumber)

	// Assert results
	s.NoError(err)
	s.Nil(result.Get(nil))

	s.mockDunningService.AssertExpectations(s.T())
	s.mockNotificationPublisher.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestHandleSubscriptionStateChanged_CancelledSubscription() {
	// Test handling cancelled subscription state change
	orgId := "org_123"
	campaignId := "campaign_456"

	campaign := testutils.MockDunningCampaign(orgId, "sub_456", "cust_789")

	stateChange := dto.SubscriptionStateChangedInput{
		NewStatus: entities.SubscriptionStatusCancelled,
	}

	cancelledCampaign := campaign
	cancelledCampaign.Status = dunning.DunningStatusCancelled

	s.mockDunningService.On("FindCampaignById", mock.Anything, orgId, campaignId).Return(
		campaign, nil)

	expectedInput := dto.CancelDunningCampaignInput{
		OrgId:  orgId,
		Id:     campaignId,
		Reason: "manual_cancel",
	}
	s.mockDunningService.On("CancelCampaign", mock.Anything, expectedInput).Return(
		cancelledCampaign, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.HandleSubscriptionStateChanged, orgId, campaignId, stateChange)

	// Assert results
	s.NoError(err)

	var resultCampaign dunning.DunningCampaign
	s.NoError(result.Get(&resultCampaign))
	s.Equal(dunning.DunningStatusCancelled, resultCampaign.Status)

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestHandleSubscriptionStateChanged_PausedSubscription() {
	// Test handling paused subscription state change
	orgId := "org_123"
	campaignId := "campaign_456"

	campaign := testutils.MockDunningCampaign(orgId, "sub_456", "cust_789")

	stateChange := dto.SubscriptionStateChangedInput{
		NewStatus: entities.SubscriptionStatusPaused,
	}

	pausedCampaign := campaign
	pausedCampaign.Status = dunning.DunningStatusPaused

	s.mockDunningService.On("FindCampaignById", mock.Anything, orgId, campaignId).Return(
		campaign, nil)

	expectedInput := dto.PauseDunningCampaignInput{
		OrgId:  orgId,
		Id:     campaignId,
		Reason: "manual_pause",
	}
	s.mockDunningService.On("PauseCampaign", mock.Anything, expectedInput).Return(
		pausedCampaign, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.HandleSubscriptionStateChanged, orgId, campaignId, stateChange)

	// Assert results
	s.NoError(err)

	var resultCampaign dunning.DunningCampaign
	s.NoError(result.Get(&resultCampaign))
	s.Equal(dunning.DunningStatusPaused, resultCampaign.Status)

	s.mockDunningService.AssertExpectations(s.T())
}
