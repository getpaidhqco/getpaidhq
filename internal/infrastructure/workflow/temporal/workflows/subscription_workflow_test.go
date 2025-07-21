package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"

	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/settings"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/infrastructure/workflow/temporal/testutils"
)

type SubscriptionWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func TestSubscriptionWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(SubscriptionWorkflowTestSuite))
}

func (s *SubscriptionWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetTestTimeout(time.Minute * 5)
}

func (s *SubscriptionWorkflowTestSuite) TearDownTest() {
	s.env.AssertExpectations(s.T())
}

func (s *SubscriptionWorkflowTestSuite) TestSubscriptionWorkflow_SuccessfulBillingCycle() {
	// Test setup
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)

	// Mock activities
	s.env.OnActivity(&activities.OrderActivities{}, "GetSubscriptionSettings", mock.Anything, subscription.OrgId).Return(
		testutils.MockSubscriptionSettings(), nil)

	s.env.OnActivity(&activities.OrderActivities{}, "ChargeCustomerForBillingPeriod", mock.Anything, subscription).Return(
		testutils.MockSuccessfulChargeResult(1000), nil)

	updatedSub := testutils.CreateUpdatedSubscription(subscription, entities.SubscriptionStatusActive)
	s.env.OnActivity(&activities.OrderActivities{}, "HandleChargeResult", mock.Anything, subscription, mock.Anything).Return(
		updatedSub, nil)

	s.env.OnActivity(&activities.OrderActivities{}, "NotifyWorkflowEnded", mock.Anything, subscription.OrgId, subscription.Id).Return(nil)

	// Execute workflow
	s.env.ExecuteWorkflow(SubscriptionWorkflow, subscription)

	// Verify workflow completed successfully
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result entities.Subscription
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(entities.SubscriptionStatusActive, result.Status)
}

func (s *SubscriptionWorkflowTestSuite) TestSubscriptionWorkflow_PausedSubscription() {
	// Test paused subscription workflow
	subscription := testutils.CreatePausedSubscription("org_123", "cust_456")

	s.env.OnActivity(&activities.OrderActivities{}, "GetSubscriptionSettings", mock.Anything, subscription.OrgId).Return(
		testutils.MockSubscriptionSettings(), nil)

	s.env.OnActivity(&activities.OrderActivities{}, "NotifyWorkflowEnded", mock.Anything, subscription.OrgId, subscription.Id).Return(nil)

	// Execute workflow
	s.env.ExecuteWorkflow(SubscriptionWorkflow, subscription)

	// Send activation signal
	activatedSub := testutils.CreateFastSubscription("org_123", "cust_456", 1000)
	s.env.SignalWorkflow("subscription.activated", activatedSub)

	// Mock charge after activation
	s.env.OnActivity(&activities.OrderActivities{}, "ChargeCustomerForBillingPeriod", mock.Anything, mock.Anything).Return(
		testutils.MockSuccessfulChargeResult(1000), nil)

	s.env.OnActivity(&activities.OrderActivities{}, "HandleChargeResult", mock.Anything, mock.Anything, mock.Anything).Return(
		activatedSub, nil)

	// Verify workflow continues after activation
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *SubscriptionWorkflowTestSuite) TestSubscriptionWorkflow_CancelledSubscription() {
	// Test cancelled subscription workflow
	subscription := testutils.CreateCancelledSubscription("org_123", "cust_456")

	s.env.OnActivity(&activities.OrderActivities{}, "GetSubscriptionSettings", mock.Anything, subscription.OrgId).Return(
		testutils.MockSubscriptionSettings(), nil)

	s.env.OnActivity(&activities.OrderActivities{}, "NotifyWorkflowEnded", mock.Anything, subscription.OrgId, subscription.Id).Return(nil)

	// Execute workflow
	s.env.ExecuteWorkflow(SubscriptionWorkflow, subscription)

	// Verify workflow ends immediately for cancelled subscription
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result entities.Subscription
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(entities.SubscriptionStatusCancelled, result.Status)
}

func (s *SubscriptionWorkflowTestSuite) TestSubscriptionWorkflow_PaymentFailureTriggersDunning() {
	// Test failed payment scenario
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)

	s.env.OnActivity(&activities.OrderActivities{}, "GetSubscriptionSettings", mock.Anything, subscription.OrgId).Return(
		testutils.MockSubscriptionSettings(), nil)

	// Mock failed charge
	s.env.OnActivity(&activities.OrderActivities{}, "ChargeCustomerForBillingPeriod", mock.Anything, subscription).Return(
		testutils.MockFailedChargeResult(1000), nil)

	// Mock subscription update to past due status
	pastDueSub := testutils.CreateUpdatedSubscription(subscription, entities.SubscriptionStatusPastDue)
	s.env.OnActivity(&activities.OrderActivities{}, "HandleChargeResult", mock.Anything, subscription, mock.Anything).Return(
		pastDueSub, nil)

	s.env.OnActivity(&activities.OrderActivities{}, "NotifyWorkflowEnded", mock.Anything, subscription.OrgId, subscription.Id).Return(nil)

	// Execute workflow
	s.env.ExecuteWorkflow(SubscriptionWorkflow, subscription)

	// Verify workflow handles failed payment
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *SubscriptionWorkflowTestSuite) TestSubscriptionWorkflow_PendingPaymentWithWebhook() {
	// Test pending payment that requires webhook
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)

	s.env.OnActivity(&activities.OrderActivities{}, "GetSubscriptionSettings", mock.Anything, subscription.OrgId).Return(
		testutils.MockSubscriptionSettings(), nil)

	// Mock pending charge
	s.env.OnActivity(&activities.OrderActivities{}, "ChargeCustomerForBillingPeriod", mock.Anything, subscription).Return(
		testutils.MockPendingChargeResult(1000), nil)

	// Execute workflow
	s.env.ExecuteWorkflow(SubscriptionWorkflow, subscription)

	// Simulate webhook signal with successful payment
	successResult := testutils.MockSuccessfulChargeResult(1000)
	s.env.SignalWorkflow("webhook-signal", successResult)

	// Mock successful charge handling
	updatedSub := testutils.CreateUpdatedSubscription(subscription, entities.SubscriptionStatusActive)
	s.env.OnActivity(&activities.OrderActivities{}, "HandleChargeResult", mock.Anything, subscription, successResult).Return(
		updatedSub, nil)

	s.env.OnActivity(&activities.OrderActivities{}, "NotifyWorkflowEnded", mock.Anything, subscription.OrgId, subscription.Id).Return(nil)

	// Verify workflow completes after webhook
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *SubscriptionWorkflowTestSuite) TestSubscriptionWorkflow_ForceUpdateSignal() {
	// Test force update signal handling
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)

	s.env.OnActivity(&activities.OrderActivities{}, "GetSubscriptionSettings", mock.Anything, subscription.OrgId).Return(
		testutils.MockSubscriptionSettings(), nil)

	// Execute workflow
	s.env.ExecuteWorkflow(SubscriptionWorkflow, subscription)

	// Send force update signal
	updatedSub := testutils.CreateSubscriptionWithNextChargeDate("org_123", "cust_456", time.Now().Add(time.Second))
	s.env.SignalWorkflow("refresh-state", updatedSub)

	// Mock activities for the refreshed cycle
	s.env.OnActivity(&activities.OrderActivities{}, "ChargeCustomerForBillingPeriod", mock.Anything, mock.Anything).Return(
		testutils.MockSuccessfulChargeResult(1000), nil)

	s.env.OnActivity(&activities.OrderActivities{}, "HandleChargeResult", mock.Anything, mock.Anything, mock.Anything).Return(
		updatedSub, nil)

	s.env.OnActivity(&activities.OrderActivities{}, "NotifyWorkflowEnded", mock.Anything, subscription.OrgId, subscription.Id).Return(nil)

	// Verify workflow processes the update
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *SubscriptionWorkflowTestSuite) TestSubscriptionWorkflow_QueryHandler() {
	// Test query handler functionality
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)

	s.env.OnActivity(&activities.OrderActivities{}, "GetSubscriptionSettings", mock.Anything, subscription.OrgId).Return(
		testutils.MockSubscriptionSettings(), nil)

	s.env.OnActivity(&activities.OrderActivities{}, "NotifyWorkflowEnded", mock.Anything, subscription.OrgId, subscription.Id).Return(nil)

	// Execute workflow
	s.env.ExecuteWorkflow(SubscriptionWorkflow, subscription)

	// Query workflow state
	encodedValue, err := s.env.QueryWorkflow("get-state")
	s.NoError(err)

	var queriedSubscription entities.Subscription
	err = encodedValue.Get(&queriedSubscription)
	s.NoError(err)
	s.Equal(subscription.Id, queriedSubscription.Id)
	s.Equal(subscription.OrgId, queriedSubscription.OrgId)
}

func (s *SubscriptionWorkflowTestSuite) TestSubscriptionWorkflow_MultipleBillingCycles() {
	// Test multiple billing cycles with fast timing
	subscription := testutils.CreateFastSubscription("org_123", "cust_456", 1000)

	// Mock settings to return fast timing
	testSettings := settings.Subscription{ReminderDays: 0}
	s.env.OnActivity(&activities.OrderActivities{}, "GetSubscriptionSettings", mock.Anything, subscription.OrgId).Return(
		testSettings, nil).Times(3) // Multiple cycles

	// Mock successful charges for multiple cycles
	chargeCounter := testutils.NewChargeAttemptCounter(0, 1000, 0) // Always succeed
	s.env.OnActivity(&activities.OrderActivities{}, "ChargeCustomerForBillingPeriod", mock.Anything, mock.Anything).Return(
		func(ctx context.Context, sub entities.Subscription) (payments.ChargeResult, error) {
			return chargeCounter.NextAttempt(), nil
		}).Times(3)

	// Mock subscription updates for each cycle
	s.env.OnActivity(&activities.OrderActivities{}, "HandleChargeResult", mock.Anything, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, sub entities.Subscription, result payments.ChargeResult) (entities.Subscription, error) {
			updated := testutils.CreateUpdatedSubscription(sub, entities.SubscriptionStatusActive)
			// Stop after 3 cycles by setting status to cancelled
			if updated.CyclesProcessed >= 3 {
				updated.Status = entities.SubscriptionStatusCancelled
			}
			return updated, nil
		}).Times(3)

	s.env.OnActivity(&activities.OrderActivities{}, "NotifyWorkflowEnded", mock.Anything, subscription.OrgId, subscription.Id).Return(nil)

	// Execute workflow
	s.env.ExecuteWorkflow(SubscriptionWorkflow, subscription)

	// Verify workflow processed multiple cycles
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result entities.Subscription
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(3, result.CyclesProcessed)
	s.Equal(3000, result.TotalRevenue) // 3 cycles * 1000
}
