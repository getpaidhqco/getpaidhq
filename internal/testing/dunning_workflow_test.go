package testing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/infrastructure/workflow/temporal/workflows"
	"payloop/internal/lib"
	"payloop/internal/testing/mocks"
)

// DunningWorkflowTestSuite is a test suite for the dunning workflow
type DunningWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env                 *testsuite.TestWorkflowEnvironment
	mockDunningService  *mocks.MockDunningService
	mockSubService      *mocks.MockSubscriptionService
	mockPubSub          *mocks.MockPubSub
	realErrorReporter   lib.ErrorReporter
	testSubscription    entities.Subscription
	testDunningCampaign dunning.DunningCampaign
}

func TestDunningWorkflowSuite(t *testing.T) {
	suite.Run(t, new(DunningWorkflowTestSuite))
}

func (s *DunningWorkflowTestSuite) SetupTest() {
	// Create the test workflow environment
	s.env = s.NewTestWorkflowEnvironment()

	// Create mock services
	s.mockDunningService = mocks.NewMockDunningService()
	s.mockSubService = mocks.NewMockSubscriptionService()
	s.mockPubSub = mocks.NewMockPubSub()
	
	// Use real error reporter
	logger := lib.GetLogger()
	s.realErrorReporter = lib.NewErrorReporter(logger)

	// Create test data
	s.setupTestData()

	// Register activities
	dunningActivities := activities.NewDunningActivities(
		s.mockDunningService,
		s.mockSubService,
		s.mockPubSub,
		s.realErrorReporter,
	)

	s.env.RegisterActivity(dunningActivities.CreateDunningCampaign)
	s.env.RegisterActivity(dunningActivities.ResolveDunningConfig)
	s.env.RegisterActivity(dunningActivities.ExecuteDunningAttempt)
	s.env.RegisterActivity(dunningActivities.UpdateCampaignWithAttemptResult)
	s.env.RegisterActivity(dunningActivities.MarkCampaignRecovered)
	s.env.RegisterActivity(dunningActivities.MarkCampaignFailed)
	s.env.RegisterActivity(dunningActivities.SendDunningCommunication)
	s.env.RegisterActivity(dunningActivities.TriggerImmediateRetry)
	s.env.RegisterActivity(dunningActivities.PauseDunningCampaign)
	s.env.RegisterActivity(dunningActivities.ResumeDunningCampaign)
	s.env.RegisterActivity(dunningActivities.CancelDunningCampaign)
	s.env.RegisterActivity(dunningActivities.HandleSubscriptionStateChanged)
	s.env.RegisterActivity(dunningActivities.ReactivateSubscription)
	s.env.RegisterActivity(dunningActivities.CancelSubscription)

	// Register workflow
	s.env.RegisterWorkflow(workflows.DunningWorkflow)
}

func (s *DunningWorkflowTestSuite) setupTestData() {
	// Create test subscription
	s.testSubscription = entities.Subscription{
		OrgId:           "org_123",
		Id:              "sub_123",
		CustomerId:      "cus_123",
		Status:          entities.SubscriptionStatusActive,
		PaymentMethodId: "pm_123",
		Currency:        "USD",
		Amount:          10000, // $100.00
		DunningActive:   false,
	}

	// Create test dunning campaign
	s.testDunningCampaign = dunning.DunningCampaign{
		OrgId:                "org_123",
		Id:                   "dun_123",
		SubscriptionId:       "sub_123",
		CustomerId:           "cus_123",
		Status:               dunning.DunningStatusActive,
		FailedAmount:         10000,
		Currency:             "USD",
		InitialFailureReason: "insufficient_funds",
		TotalAttempts:        0,
		StartedAt:            time.Now(),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

func (s *DunningWorkflowTestSuite) TearDownTest() {
	s.env.AssertExpectations(s.T())
}

func (s *DunningWorkflowTestSuite) TestDunningWorkflow_SuccessfulRecoveryOnFirstRetry() {
	// Setup mock expectations
	
	// 1. CreateDunningCampaign
	s.mockDunningService.On("CreateCampaign", mock.Anything, mock.Anything).
		Return(s.testDunningCampaign, nil)
	
	// 2. ResolveDunningConfig
	dunningConfig := dunning.DefaultDunningConfig()
	dunningConfig.ImmediateRetries.Enabled = true
	dunningConfig.ImmediateRetries.MaxAttempts = 3
	dunningConfig.ImmediateRetries.FailureTypes = []string{"insufficient_funds"}
	dunningConfig.ImmediateRetries.Intervals = []string{"5m", "30m", "2h"}
	
	s.mockDunningService.On("ListConfigurations", mock.Anything, mock.Anything, mock.Anything).
		Return([]dunning.DunningConfiguration{}, 0, nil)
	
	// 3. ExecuteDunningAttempt - successful on first retry
	successfulAttempt := dunning.DunningAttempt{
		OrgId:             "org_123",
		Id:                "att_123",
		DunningCampaignId: "dun_123",
		SubscriptionId:    "sub_123",
		AttemptNumber:     1,
		AttemptType:       dunning.DunningAttemptTypeImmediate,
		Status:            payments.PaymentStatusSucceeded,
		Amount:            10000,
		Currency:          "USD",
		PaymentMethodId:   "pm_123",
		AttemptedAt:       time.Now(),
		CompletedAt:       time.Now(),
	}
	
	s.mockDunningService.On("TriggerManualAttempt", mock.Anything, mock.Anything).
		Return(successfulAttempt, nil)
	
	// 4. UpdateCampaignWithAttemptResult
	recoveredCampaign := s.testDunningCampaign
	recoveredCampaign.Status = dunning.DunningStatusRecovered
	recoveredCampaign.TotalAttempts = 1
	recoveredCampaign.RecoveredAt = time.Now()
	
	s.mockDunningService.On("FindCampaignById", mock.Anything, "org_123", "dun_123").
		Return(s.testDunningCampaign, nil)
	
	s.mockDunningService.On("UpdateCampaign", mock.Anything, "org_123", mock.Anything).
		Return(recoveredCampaign, nil)
	
	// 5. Publish event
	s.mockPubSub.On("Publish", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	
	// Execute workflow
	workflowInput := workflows.DunningWorkflowInput{
		OrgId:                "org_123",
		SubscriptionId:       "sub_123",
		CustomerId:           "cus_123",
		FailedAmount:         10000,
		Currency:             "USD",
		InitialFailureReason: "insufficient_funds",
		PaymentResult: payments.ChargeResult{
			Status:  payments.PaymentStatusFailed,
		},
	}
	
	s.env.ExecuteWorkflow(workflows.DunningWorkflow, workflowInput)
	
	// Verify workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	
	// Verify workflow result
	var result dunning.DunningCampaign
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(dunning.DunningStatusRecovered, result.Status)
	s.Equal(1, result.TotalAttempts)
}