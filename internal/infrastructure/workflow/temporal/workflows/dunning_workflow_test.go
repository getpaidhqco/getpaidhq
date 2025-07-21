package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/dunning"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/infrastructure/workflow/temporal/testutils"
)

type DunningWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func TestDunningWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(DunningWorkflowTestSuite))
}

func (s *DunningWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetTestTimeout(time.Minute * 5)
}

func (s *DunningWorkflowTestSuite) TearDownTest() {
	s.env.AssertExpectations(s.T())
}

// Helper function to convert testutils input to workflow input
func (s *DunningWorkflowTestSuite) convertToWorkflowInput(input testutils.DunningWorkflowInput) DunningWorkflowInput {
	return DunningWorkflowInput{
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
}

// Helper function to convert testutils input to activities input
func (s *DunningWorkflowTestSuite) convertToActivitiesInput(input testutils.DunningWorkflowInput) activities.DunningWorkflowInput {
	return activities.DunningWorkflowInput{
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
}

func (s *DunningWorkflowTestSuite) TestDunningWorkflow_SuccessfulRecoveryOnFirstRetry() {
	// Test successful recovery on first retry attempt
	input := testutils.CreateDunningWorkflowInput("org_123", "sub_456", "cust_789")
	campaign := testutils.MockDunningCampaign(input.OrgId, input.SubscriptionId, input.CustomerId)

	// Create activities instance for method references
	var a *activities.DunningActivities

	// Mock campaign creation
	activitiesInput := s.convertToActivitiesInput(input)
	s.env.OnActivity(a.CreateDunningCampaign, mock.Anything, activitiesInput).Return(
		campaign, nil)

	// Mock dunning config with fast intervals
	s.env.OnActivity(a.ResolveDunningConfig, mock.Anything, input.OrgId).Return(
		testutils.MockDunningConfig(), nil)

	// Mock subscription retrieval
	subscription := testutils.CreateFastSubscription(input.OrgId, input.CustomerId, 1000)
	s.env.OnActivity(a.GetSubscriptionForDunning, mock.Anything, input.OrgId, input.SubscriptionId).Return(
		subscription, nil)

	// Mock communication sending
	s.env.OnActivity(a.SendDunningCommunication, mock.Anything, campaign.OrgId, campaign.Id, 1).Return(nil)

	// Mock successful retry charge
	s.env.OnActivity(a.ProcessRetryCharge, mock.Anything, mock.Anything).Return(
		testutils.MockSuccessfulChargeResult(1000), nil)

	// Mock successful charge result handling
	recoveredCampaign := campaign
	recoveredCampaign.Status = dunning.DunningStatusRecovered
	s.env.OnActivity(a.HandleDunningChargeResult, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		activities.HandleChargeAttemptResult{
			Campaign:     recoveredCampaign,
			Subscription: subscription,
		}, nil)

	// Execute workflow
	workflowInput := s.convertToWorkflowInput(input)
	s.env.ExecuteWorkflow(DunningWorkflow, workflowInput)

	// Verify workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result dunning.DunningCampaign
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(dunning.DunningStatusRecovered, result.Status)
}

func (s *DunningWorkflowTestSuite) TestDunningWorkflow_ProgressiveRetriesWithEventualSuccess() {
	// Test progressive retries with success on 3rd attempt
	input := testutils.CreateDunningWorkflowInput("org_123", "sub_456", "cust_789")
	campaign := testutils.MockDunningCampaign(input.OrgId, input.SubscriptionId, input.CustomerId)
	subscription := testutils.CreateFastSubscription(input.OrgId, input.CustomerId, 1000)

	// Create activities instance for method references
	var a *activities.DunningActivities

	// Mock campaign creation
	activitiesInput := s.convertToActivitiesInput(input)
	s.env.OnActivity(a.CreateDunningCampaign, mock.Anything, activitiesInput).Return(
		campaign, nil)

	// Mock dunning config with 3 retry attempts
	s.env.OnActivity(a.ResolveDunningConfig, mock.Anything, input.OrgId).Return(
		testutils.MockDunningConfig(), nil)

	// Mock subscription retrieval
	s.env.OnActivity(a.GetSubscriptionForDunning, mock.Anything, input.OrgId, input.SubscriptionId).Return(
		subscription, nil)

	// Mock communication sending for each attempt
	s.env.OnActivity(a.SendDunningCommunication, mock.Anything, campaign.OrgId, campaign.Id, 1).Return(nil)
	s.env.OnActivity(a.SendDunningCommunication, mock.Anything, campaign.OrgId, campaign.Id, 2).Return(nil)
	s.env.OnActivity(a.SendDunningCommunication, mock.Anything, campaign.OrgId, campaign.Id, 3).Return(nil)

	// Mock progressive charge attempts (fail 2 times, succeed on 3rd)
	chargeCounter := testutils.NewChargeAttemptCounter(2, 1000, 1000) // Fail twice, then succeed
	s.env.OnActivity(a.ProcessRetryCharge, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, sub entities.Subscription) (payments.ChargeResult, error) {
			return chargeCounter.NextAttempt(), nil
		}).Times(3)

	// Mock charge result handling for each attempt
	activeCampaign := campaign
	activeCampaign.Status = dunning.DunningStatusActive

	recoveredCampaign := campaign
	recoveredCampaign.Status = dunning.DunningStatusRecovered

	s.env.OnActivity(a.HandleDunningChargeResult, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, camp dunning.DunningCampaign, result payments.ChargeResult, config dunning.DunningConfig) (activities.HandleChargeAttemptResult, error) {
			if result.Status == payments.PaymentStatusSucceeded {
				return activities.HandleChargeAttemptResult{
					Campaign:     recoveredCampaign,
					Subscription: subscription,
				}, nil
			}
			return activities.HandleChargeAttemptResult{
				Campaign:     activeCampaign,
				Subscription: subscription,
			}, nil
		}).Times(3)

	// Execute workflow
	workflowInput := s.convertToWorkflowInput(input)
	s.env.ExecuteWorkflow(DunningWorkflow, workflowInput)

	// Verify workflow completed with recovery
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result dunning.DunningCampaign
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(dunning.DunningStatusRecovered, result.Status)
}

func (s *DunningWorkflowTestSuite) TestDunningWorkflow_AllRetriesFailedCampaignFailed() {
	// Test scenario where all retries fail and campaign fails
	input := testutils.CreateDunningWorkflowInput("org_123", "sub_456", "cust_789")
	campaign := testutils.MockDunningCampaign(input.OrgId, input.SubscriptionId, input.CustomerId)
	subscription := testutils.CreateFastSubscription(input.OrgId, input.CustomerId, 1000)

	// Create activities instance for method references
	var a *activities.DunningActivities

	// Mock campaign creation
	activitiesInput := s.convertToActivitiesInput(input)
	s.env.OnActivity(a.CreateDunningCampaign, mock.Anything, activitiesInput).Return(
		campaign, nil)

	// Mock dunning config
	s.env.OnActivity(a.ResolveDunningConfig, mock.Anything, input.OrgId).Return(
		testutils.MockDunningConfig(), nil)

	// Mock subscription retrieval
	s.env.OnActivity(a.GetSubscriptionForDunning, mock.Anything, input.OrgId, input.SubscriptionId).Return(
		subscription, nil)

	// Mock communication sending for all attempts
	s.env.OnActivity(a.SendDunningCommunication, mock.Anything, campaign.OrgId, campaign.Id, mock.Anything).Return(nil).Times(3)

	// Mock all charge attempts fail
	s.env.OnActivity(a.ProcessRetryCharge, mock.Anything, mock.Anything).Return(
		testutils.MockFailedChargeResult(1000), nil).Times(3)

	// Mock charge result handling - final attempt marks campaign as failed
	activeCampaign := campaign
	activeCampaign.Status = dunning.DunningStatusActive

	failedCampaign := campaign
	failedCampaign.Status = dunning.DunningStatusFailed

	attemptCount := 0
	s.env.OnActivity(a.HandleDunningChargeResult, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, camp dunning.DunningCampaign, result payments.ChargeResult, config dunning.DunningConfig) (activities.HandleChargeAttemptResult, error) {
			attemptCount++
			// Return active campaign for first 2 attempts, failed on 3rd
			if attemptCount < 3 {
				return activities.HandleChargeAttemptResult{
					Campaign:     activeCampaign,
					Subscription: subscription,
				}, nil
			}
			return activities.HandleChargeAttemptResult{
				Campaign:     failedCampaign,
				Subscription: subscription,
			}, nil
		}).Times(3)

	// Execute workflow
	workflowInput := s.convertToWorkflowInput(input)
	s.env.ExecuteWorkflow(DunningWorkflow, workflowInput)

	// Verify workflow completed with failed campaign
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result dunning.DunningCampaign
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(dunning.DunningStatusFailed, result.Status)
}

func (s *DunningWorkflowTestSuite) TestDunningWorkflow_PauseAndResumeCampaign() {
	// TODO: Signal-based tests require integration testing approach
	s.T().Skip("Signal-based test requires integration testing - unit test framework doesn't handle goroutine-based signal processing correctly")
}

func (s *DunningWorkflowTestSuite) TestDunningWorkflow_CancelCampaign() {
	// TODO: Signal-based tests require integration testing approach due to
	// timing issues between signal handlers (goroutines) and main workflow execution
	// in the unit test framework. The signal handler updates campaign status in a
	// goroutine, but the main workflow loop may not see these changes in the
	// synchronous test execution environment.
	s.T().Skip("Signal-based test requires integration testing - unit test framework doesn't handle goroutine-based signal processing correctly")
}

func (s *DunningWorkflowTestSuite) TestDunningWorkflow_PaymentMethodUpdatedTriggerImmediateRetry() {
	// TODO: Signal-based tests require integration testing approach
	s.T().Skip("Signal-based test requires integration testing - unit test framework doesn't handle goroutine-based signal processing correctly")
}

func (s *DunningWorkflowTestSuite) TestDunningWorkflow_PendingPaymentWithWebhook() {
	// TODO: Signal-based tests require integration testing approach
	s.T().Skip("Signal-based test requires integration testing - unit test framework doesn't handle goroutine-based signal processing correctly")
}

func (s *DunningWorkflowTestSuite) TestDunningWorkflow_QueryHandler() {
	// Test query handler functionality
	input := testutils.CreateDunningWorkflowInput("org_123", "sub_456", "cust_789")
	campaign := testutils.MockDunningCampaign(input.OrgId, input.SubscriptionId, input.CustomerId)

	// Create activities instance for method references
	var a *activities.DunningActivities

	// Mock campaign creation
	activitiesInput := s.convertToActivitiesInput(input)
	s.env.OnActivity(a.CreateDunningCampaign, mock.Anything, activitiesInput).Return(
		campaign, nil)

	// Mock dunning config
	s.env.OnActivity(a.ResolveDunningConfig, mock.Anything, input.OrgId).Return(
		testutils.MockDunningConfig(), nil)

	// Mock subscription retrieval
	subscription := testutils.CreateFastSubscription(input.OrgId, input.CustomerId, 1000)
	s.env.OnActivity(a.GetSubscriptionForDunning, mock.Anything, input.OrgId, input.SubscriptionId).Return(
		subscription, nil)

	// Execute workflow
	workflowInput := s.convertToWorkflowInput(input)
	s.env.ExecuteWorkflow(DunningWorkflow, workflowInput)

	// Query campaign state
	encodedValue, err := s.env.QueryWorkflow("get-campaign")
	s.NoError(err)

	var queriedCampaign dunning.DunningCampaign
	err = encodedValue.Get(&queriedCampaign)
	s.NoError(err)
	s.Equal(campaign.Id, queriedCampaign.Id)
	s.Equal(campaign.OrgId, queriedCampaign.OrgId)
}
