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
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/settings"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/infrastructure/workflow/temporal/testutils"
	"payloop/internal/lib"
)

// Mock interfaces for dunning activities testing
type DunningMockService struct {
	mock.Mock
}

func (m *DunningMockService) CreateCampaign(ctx context.Context, input dto.CreateDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *DunningMockService) FindCampaignById(ctx context.Context, orgId, campaignId string) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, orgId, campaignId)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *DunningMockService) ListConfigurations(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningConfiguration, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]dunning.DunningConfiguration), args.Get(1).(int), args.Error(2)
}

// Add missing DunningService methods
func (m *DunningMockService) ListCampaigns(ctx context.Context, orgId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]dunning.DunningCampaign), args.Get(1).(int), args.Error(2)
}

func (m *DunningMockService) ListCampaignsBySubscription(ctx context.Context, orgId string, subscriptionId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	args := m.Called(ctx, orgId, subscriptionId, pagination)
	return args.Get(0).([]dunning.DunningCampaign), args.Get(1).(int), args.Error(2)
}

func (m *DunningMockService) ListCampaignsByCustomer(ctx context.Context, orgId string, customerId string, pagination request.Pagination) ([]dunning.DunningCampaign, int, error) {
	args := m.Called(ctx, orgId, customerId, pagination)
	return args.Get(0).([]dunning.DunningCampaign), args.Get(1).(int), args.Error(2)
}

func (m *DunningMockService) UpdateCampaign(ctx context.Context, orgId string, campaign dunning.DunningCampaign) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, orgId, campaign)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *DunningMockService) ListAttemptsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningAttempt, int, error) {
	args := m.Called(ctx, orgId, campaignId, pagination)
	return args.Get(0).([]dunning.DunningAttempt), args.Get(1).(int), args.Error(2)
}

func (m *DunningMockService) TriggerChargeAttempt(ctx context.Context, input dto.TriggerAttemptInput) (dunning.DunningAttempt, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningAttempt), args.Error(1)
}

func (m *DunningMockService) ListCommunicationsByCampaign(ctx context.Context, orgId string, campaignId string, pagination request.Pagination) ([]dunning.DunningCommunication, int, error) {
	args := m.Called(ctx, orgId, campaignId, pagination)
	return args.Get(0).([]dunning.DunningCommunication), args.Get(1).(int), args.Error(2)
}

func (m *DunningMockService) CreatePaymentUpdateToken(ctx context.Context, input dto.CreatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

func (m *DunningMockService) VerifyPaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, orgId, tokenId)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

func (m *DunningMockService) ActivatePaymentUpdateToken(ctx context.Context, input dto.ActivatePaymentUpdateTokenInput) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

func (m *DunningMockService) RevokePaymentUpdateToken(ctx context.Context, orgId string, tokenId string) (dunning.PaymentUpdateToken, error) {
	args := m.Called(ctx, orgId, tokenId)
	return args.Get(0).(dunning.PaymentUpdateToken), args.Error(1)
}

func (m *DunningMockService) CreateConfiguration(ctx context.Context, input dto.CreateDunningConfigurationInput) (dunning.DunningConfiguration, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningConfiguration), args.Error(1)
}

func (m *DunningMockService) GetConfiguration(ctx context.Context, orgId string, id string) (dunning.DunningConfiguration, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(dunning.DunningConfiguration), args.Error(1)
}

func (m *DunningMockService) UpdateConfiguration(ctx context.Context, input dto.UpdateDunningConfigurationInput) (dunning.DunningConfiguration, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningConfiguration), args.Error(1)
}

func (m *DunningMockService) GetCustomerDunningHistory(ctx context.Context, orgId string, customerId string) (dunning.CustomerDunningHistory, error) {
	args := m.Called(ctx, orgId, customerId)
	return args.Get(0).(dunning.CustomerDunningHistory), args.Error(1)
}

func (m *DunningMockService) PauseCampaign(ctx context.Context, input dto.PauseDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *DunningMockService) ResumeCampaign(ctx context.Context, input dto.ResumeDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *DunningMockService) CancelCampaign(ctx context.Context, input dto.CancelDunningCampaignInput) (dunning.DunningCampaign, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dunning.DunningCampaign), args.Error(1)
}

func (m *DunningMockService) HandleChargeResult(ctx context.Context, campaign dunning.DunningCampaign, result payments.ChargeResult, config dunning.DunningConfig) (dto.HandleChargeResultResponse, error) {
	args := m.Called(ctx, campaign, result, config)
	return args.Get(0).(dto.HandleChargeResultResponse), args.Error(1)
}

type DunningMockSubscriptionService struct {
	mock.Mock
}

func (m *DunningMockSubscriptionService) CreateSubscriptionsForOrder(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
	args := m.Called(ctx, orgId, orderId)
	return args.Get(0).([]entities.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) Create(ctx context.Context, orgId string, input dto.CreateSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, orgId, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	args := m.Called(ctx, orgId, id)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) Activate(ctx context.Context, input subscriptions.ActivateSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) PauseSubscription(ctx context.Context, input subscriptions.PauseSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Subscription, int, error) {
	args := m.Called(ctx, orgId, pagination)
	return args.Get(0).([]entities.Subscription), args.Get(1).(int), args.Error(2)
}

func (m *DunningMockSubscriptionService) FindSubscriptionPayments(ctx context.Context, pk entities.EntityKey, pagination request.Pagination) ([]entities.Payment, int, error) {
	args := m.Called(ctx, pk, pagination)
	return args.Get(0).([]entities.Payment), args.Get(1).(int), args.Error(2)
}

func (m *DunningMockSubscriptionService) ResumeSubscription(ctx context.Context, input subscriptions.ResumeSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) CancelSubscription(ctx context.Context, input subscriptions.CancelSubscriptionInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) ChangeSubscriptionPlan(ctx context.Context, input subscriptions.ChangePlanInput) (*entities.Subscription, *entities.SubscriptionPlanChange, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*entities.Subscription), args.Get(1).(*entities.SubscriptionPlanChange), args.Error(2)
}

func (m *DunningMockSubscriptionService) UpdateBillingAnchor(ctx context.Context, input dto.UpdateBillingAnchorInput) (dto.UpdateBillingAnchorResult, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(dto.UpdateBillingAnchorResult), args.Error(1)
}

func (m *DunningMockSubscriptionService) GetSubscriptionCustomer(ctx context.Context, subscription entities.Subscription) (entities.Customer, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.Customer), args.Error(1)
}

func (m *DunningMockSubscriptionService) GetSubscriptionPaymentMethod(ctx context.Context, subscription entities.Subscription) (entities.SecurePaymentMethod, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(entities.SecurePaymentMethod), args.Error(1)
}

func (m *DunningMockSubscriptionService) HandleSubscriptionChargeSuccess(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) HandleSubscriptionChargeFailure(ctx context.Context, input subscriptions.SubscriptionChargeInput) (entities.Subscription, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(entities.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) GetOrgSubscriptionSettings(ctx context.Context, orgId string) (settings.Subscription, error) {
	args := m.Called(ctx, orgId)
	return args.Get(0).(settings.Subscription), args.Error(1)
}

func (m *DunningMockSubscriptionService) ProcessSubscriptionCharge(ctx context.Context, subscription entities.Subscription) (payments.ChargeResult, error) {
	args := m.Called(ctx, subscription)
	return args.Get(0).(payments.ChargeResult), args.Error(1)
}

type DunningMockNotificationPublisher struct {
	mock.Mock
}

func (m *DunningMockNotificationPublisher) Publish(orgId string, topic string, message interface{}) error {
	args := m.Called(orgId, topic, message)
	return args.Error(0)
}

func (m *DunningMockNotificationPublisher) Subscribe(topic string, handler func(topic string, data []byte)) (events.Subscription, error) {
	args := m.Called(topic, handler)
	return args.Get(0).(events.Subscription), args.Error(1)
}

type DunningMockErrorReporter struct {
	mock.Mock
}

func (m *DunningMockErrorReporter) ReportError(ctx interface{}, err error, data map[string]interface{}) {
	m.Called(ctx, err, data)
}

type DunningMockTransactionService struct {
	mock.Mock
}

func (m *DunningMockTransactionService) WithTransaction(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
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
	mockDunningService        *DunningMockService
	mockSubscriptionService   *DunningMockSubscriptionService
	mockNotificationPublisher *DunningMockNotificationPublisher
	mockErrorReporter         *DunningMockErrorReporter
	mockTransactionService    *DunningMockTransactionService
}

func TestDunningActivitiesTestSuite(t *testing.T) {
	suite.Run(t, new(DunningActivitiesTestSuite))
}

func (s *DunningActivitiesTestSuite) SetupTest() {
	s.env = s.NewTestActivityEnvironment()

	// Create mocks
	s.mockDunningService = new(DunningMockService)
	s.mockSubscriptionService = new(DunningMockSubscriptionService)
	s.mockNotificationPublisher = new(DunningMockNotificationPublisher)
	s.mockErrorReporter = new(DunningMockErrorReporter)
	s.mockTransactionService = new(DunningMockTransactionService)

	// Create activities instance with mocks  
	activities := NewDunningActivities(
		s.mockDunningService,
		s.mockSubscriptionService,
		s.mockNotificationPublisher,
		lib.ErrorReporter{}, // Use the actual struct
		s.mockTransactionService,
	)
	s.activities = &activities

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
	// Verify all mocks were called as expected
	s.mockDunningService.AssertExpectations(s.T())
	s.mockSubscriptionService.AssertExpectations(s.T())
	s.mockNotificationPublisher.AssertExpectations(s.T())
	s.mockErrorReporter.AssertExpectations(s.T())
	s.mockTransactionService.AssertExpectations(s.T())
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

	// Convert testutils input to actual workflow input
	workflowInput := DunningWorkflowInput{
		OrgId:                input.OrgId,
		SubscriptionId:       input.SubscriptionId,
		CustomerId:           input.CustomerId,
		FailedAmount:         input.FailedAmount,
		Currency:             input.Currency,
		InitialFailureReason: input.InitialFailureReason,
		ParentWorkflowId:     input.ParentWorkflowId,
		PaymentResult:        input.PaymentResult,
		Metadata:             input.Metadata,
	}

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.CreateDunningCampaign, workflowInput)

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

	// Convert testutils input to actual workflow input
	workflowInput := DunningWorkflowInput{
		OrgId:                input.OrgId,
		SubscriptionId:       input.SubscriptionId,
		CustomerId:           input.CustomerId,
		FailedAmount:         input.FailedAmount,
		Currency:             input.Currency,
		InitialFailureReason: input.InitialFailureReason,
		ParentWorkflowId:     input.ParentWorkflowId,
		PaymentResult:        input.PaymentResult,
		Metadata:             input.Metadata,
	}

	// Execute activity
	_, err := s.env.ExecuteActivity(s.activities.CreateDunningCampaign, workflowInput)

	// Assert error
	s.Error(err)
	s.Contains(err.Error(), "campaign creation failed")

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestResolveDunningConfig_Success() {
	// Test successful config resolution
	orgId := "org_123"

	configs := []dunning.DunningConfiguration{
		{
			OrgId:    orgId,
			Priority: 1,
			Status:   dunning.ConfigStatusActive,
		},
	}

	s.mockDunningService.On("ListConfigurations", mock.Anything, orgId, mock.Anything).Return(
		configs, 1, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.ResolveDunningConfig, orgId)

	// Assert results
	s.NoError(err)

	var config dunning.DunningConfig
	s.NoError(result.Get(&config))
	// Should return default config since custom config parsing is not implemented
	s.Equal(5, config.ProgressiveRetries.MaxAttempts) // Default config has 5 max attempts

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestProcessRetryCharge_Success() {
	// Test successful charge retry
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	expectedResult := testutils.MockSuccessfulChargeResult(1000)

	s.mockSubscriptionService.On("ProcessSubscriptionCharge", mock.Anything, mock.Anything).Return(
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

func (s *DunningActivitiesTestSuite) TestSendDunningCommunication_Success() {
	// Test successful communication sending
	orgId := "org_123"
	campaignId := "campaign_456"
	attemptNumber := 1

	campaign := testutils.MockDunningCampaign(orgId, "sub_456", "cust_789")

	s.mockDunningService.On("FindCampaignById", mock.Anything, orgId, campaignId).Return(
		campaign, nil)

	s.mockNotificationPublisher.On("Publish", orgId, topic.DunningCommunicationSent, mock.Anything).Return(nil)

	// Execute activity
	_, err := s.env.ExecuteActivity(s.activities.SendDunningCommunication, orgId, campaignId, attemptNumber)

	// Assert results
	s.NoError(err)
	// Test passed if no error occurred

	s.mockDunningService.AssertExpectations(s.T())
	s.mockNotificationPublisher.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestPauseDunningCampaign_Success() {
	// Test successful campaign pause
	orgId := "org_123"
	campaignId := "campaign_456"

	expectedCampaign := testutils.MockDunningCampaign(orgId, "sub_123", "cust_456")
	expectedCampaign.Status = dunning.DunningStatusPaused

	s.mockDunningService.On("PauseCampaign", mock.Anything, mock.MatchedBy(func(input dto.PauseDunningCampaignInput) bool {
		return input.OrgId == orgId && input.Id == campaignId && input.Reason == "manual_pause"
	})).Return(expectedCampaign, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.PauseDunningCampaign, orgId, campaignId)

	// Assert results
	s.NoError(err)

	var campaign dunning.DunningCampaign
	s.NoError(result.Get(&campaign))
	s.Equal(dunning.DunningStatusPaused, campaign.Status)

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestResumeDunningCampaign_Success() {
	// Test successful campaign resume
	orgId := "org_123"
	campaignId := "campaign_456"

	expectedCampaign := testutils.MockDunningCampaign(orgId, "sub_123", "cust_456")
	expectedCampaign.Status = dunning.DunningStatusActive

	s.mockDunningService.On("ResumeCampaign", mock.Anything, mock.MatchedBy(func(input dto.ResumeDunningCampaignInput) bool {
		return input.OrgId == orgId && input.Id == campaignId && input.Reason == "manual_resume"
	})).Return(expectedCampaign, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.ResumeDunningCampaign, orgId, campaignId)

	// Assert results
	s.NoError(err)

	var campaign dunning.DunningCampaign
	s.NoError(result.Get(&campaign))
	s.Equal(dunning.DunningStatusActive, campaign.Status)

	s.mockDunningService.AssertExpectations(s.T())
}

func (s *DunningActivitiesTestSuite) TestCancelDunningCampaign_Success() {
	// Test successful campaign cancellation
	orgId := "org_123"
	campaignId := "campaign_456"

	expectedCampaign := testutils.MockDunningCampaign(orgId, "sub_123", "cust_456")
	expectedCampaign.Status = dunning.DunningStatusCancelled

	s.mockDunningService.On("CancelCampaign", mock.Anything, mock.MatchedBy(func(input dto.CancelDunningCampaignInput) bool {
		return input.OrgId == orgId && input.Id == campaignId && input.Reason == "manual_cancel"
	})).Return(expectedCampaign, nil)

	// Execute activity
	result, err := s.env.ExecuteActivity(s.activities.CancelDunningCampaign, orgId, campaignId)

	// Assert results
	s.NoError(err)

	var campaign dunning.DunningCampaign
	s.NoError(result.Get(&campaign))
	s.Equal(dunning.DunningStatusCancelled, campaign.Status)

	s.mockDunningService.AssertExpectations(s.T())
}