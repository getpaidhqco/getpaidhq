package workflows

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// These exercise the workflow orchestration (activity sequencing, result
// mapping, error propagation, timer waits) with the Temporal test environment,
// mocking the activities so no real services/DB are needed. The activity
// structs register with nil deps — they're never invoked, only mocked.

func TestPaymentRefundedWorkflow(t *testing.T) {
	t.Run("success returns the processed payment", func(t *testing.T) {
		var ts testsuite.WorkflowTestSuite
		env := ts.NewTestWorkflowEnvironment()
		oa := &activities.OrderActivities{}
		env.RegisterActivity(oa)
		env.OnActivity(oa.HandlePaymentRefundedEvent, mock.Anything, mock.Anything).
			Return(domain.Payment{Id: "pmt_1"}, nil)

		env.ExecuteWorkflow(PaymentRefunded, domain.PaymentWebhookContext{OrgId: "org_1", OrderId: "ord_1"})

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())
		var res port.WorkflowResult
		require.NoError(t, env.GetWorkflowResult(&res))
		assert.True(t, res.Success)
	})

	t.Run("activity failure fails the workflow", func(t *testing.T) {
		var ts testsuite.WorkflowTestSuite
		env := ts.NewTestWorkflowEnvironment()
		oa := &activities.OrderActivities{}
		env.RegisterActivity(oa)
		env.OnActivity(oa.HandlePaymentRefundedEvent, mock.Anything, mock.Anything).
			Return(domain.Payment{}, errors.New("downstream boom"))

		env.ExecuteWorkflow(PaymentRefunded, domain.PaymentWebhookContext{OrgId: "org_1"})

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())
	})
}

func TestPaymentSuccessWorkflow_NoSubscriptions(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	oa := &activities.OrderActivities{}
	env.RegisterActivity(oa)
	env.OnActivity(oa.CompleteOrder, mock.Anything, mock.Anything).Return(domain.Order{Id: "ord_1"}, nil)
	env.OnActivity(oa.GetOrderSubscriptions, mock.Anything, mock.Anything, mock.Anything).Return([]domain.Subscription{}, nil)

	env.ExecuteWorkflow(PaymentSuccessWorkflow, PaymentSuccessInput{
		PaymentContext: domain.PaymentWebhookContext{OrgId: "org_1", OrderId: "ord_1"},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var res port.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&res))
	assert.True(t, res.Success)
	assert.Equal(t, "no subscriptions for order", res.Message)
}

func TestPaymentSuccessWorkflow_CompleteOrderFailurePropagates(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	oa := &activities.OrderActivities{}
	env.RegisterActivity(oa)
	env.OnActivity(oa.CompleteOrder, mock.Anything, mock.Anything).Return(domain.Order{}, errors.New("order gone"))

	env.ExecuteWorkflow(PaymentSuccessWorkflow, PaymentSuccessInput{
		PaymentContext: domain.PaymentWebhookContext{OrgId: "org_1", OrderId: "ord_1"},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

func TestOutgoingWebhookWorkflow(t *testing.T) {
	t.Run("delivers successfully", func(t *testing.T) {
		var ts testsuite.WorkflowTestSuite
		env := ts.NewTestWorkflowEnvironment()
		wa := &activities.OutgoingWebhookActivities{}
		env.RegisterActivity(wa)
		env.OnActivity(wa.SendWebhook, mock.Anything, mock.Anything).Return(nil)

		env.ExecuteWorkflow(OutgoingWebhookWorkflow, port.OutgoingWebhookPayload{})

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())
		var res port.WorkflowResult
		require.NoError(t, env.GetWorkflowResult(&res))
		assert.True(t, res.Success)
	})

	t.Run("delivery failure surfaces as a workflow error", func(t *testing.T) {
		var ts testsuite.WorkflowTestSuite
		env := ts.NewTestWorkflowEnvironment()
		wa := &activities.OutgoingWebhookActivities{}
		env.RegisterActivity(wa)
		env.OnActivity(wa.SendWebhook, mock.Anything, mock.Anything).Return(errors.New("503"))

		env.ExecuteWorkflow(OutgoingWebhookWorkflow, port.OutgoingWebhookPayload{})

		require.True(t, env.IsWorkflowCompleted())
		require.Error(t, env.GetWorkflowError())
	})
}

func TestDunningAttemptWorkflow(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	da := &activities.DunningActivities{}
	env.RegisterActivity(da)
	env.OnActivity(da.ExecuteAttempt, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(domain.DunningAttempt{Id: "att_1", Status: domain.PaymentStatusSucceeded}, nil)

	env.ExecuteWorkflow(DunningAttemptWorkflow, DunningAttemptInput{
		OrgId: "org_1", CampaignId: "dc_1", AttemptNumber: 1, AttemptType: domain.DunningAttemptTypeProgressive,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var att domain.DunningAttempt
	require.NoError(t, env.GetWorkflowResult(&att))
	assert.Equal(t, "att_1", att.Id)
}

func TestBillingCycleWorkflow(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	oa := &activities.OrderActivities{}
	env.RegisterActivity(oa)
	env.OnActivity(oa.ChargeCustomerForBillingPeriod, mock.Anything, mock.Anything).
		Return(domain.ChargeResult{Status: domain.PaymentStatusSucceeded, Amount: 1000}, nil)

	env.ExecuteWorkflow(BillingCycleWorkflow, BillingCycleInput{Subscription: domain.Subscription{Id: "sub_1", Amount: 1000}})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var res domain.ChargeResult
	require.NoError(t, env.GetWorkflowResult(&res))
	assert.Equal(t, domain.PaymentStatusSucceeded, res.Status)
}

func TestSubscriptionChargeReminderWorkflow_WaitsThenSends(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	oa := &activities.OrderActivities{}
	env.RegisterActivity(oa)
	// The reminder is an hour out; the test env auto-advances the timer so this
	// completes instantly. Assert the reminder activity actually fires.
	env.OnActivity(oa.ProcessReminderEvent, mock.Anything, mock.Anything).Return(nil).Once()

	env.ExecuteWorkflow(SubscriptionChargeReminder, ReminderInput{
		Subscription: domain.Subscription{Id: "sub_1"},
		ReminderAt:   env.Now().Add(time.Hour),
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestDunningRunnerWorkflow(t *testing.T) {
	t.Run("recovered during immediate retries", func(t *testing.T) {
		var ts testsuite.WorkflowTestSuite
		env := ts.NewTestWorkflowEnvironment()
		da := &activities.DunningActivities{}
		env.RegisterActivity(da)

		config := domain.DunningConfig{
			ImmediateRetries: domain.ImmediateRetriesConfig{
				Enabled:      true,
				MaxAttempts:  2,
				Intervals:    []string{"1m", "5m"},
				FailureTypes: []string{"rate_limit"},
			},
		}

		env.OnActivity(da.LoadConfigForCampaign, mock.Anything, "org_1", "dc_1").Return(config, nil)

		// Mock child workflow for attempt 1
		env.OnWorkflow(DunningAttemptWorkflow, mock.Anything, mock.Anything).Return(domain.DunningAttempt{
			Status: domain.PaymentStatusFailed,
		}, nil).Once()

		// Campaign stays active
		env.OnActivity(da.UpdateCampaignWithAttemptResult, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(domain.DunningCampaign{Status: domain.DunningStatusActive}, nil).Once()

		// Mock child workflow for attempt 2 - Succeeded
		env.OnWorkflow(DunningAttemptWorkflow, mock.Anything, mock.Anything).Return(domain.DunningAttempt{
			Status: domain.PaymentStatusSucceeded,
		}, nil).Once()

		// Campaign recovered
		env.OnActivity(da.UpdateCampaignWithAttemptResult, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(domain.DunningCampaign{Status: domain.DunningStatusRecovered}, nil).Once()

		env.ExecuteWorkflow(DunningRunnerWorkflow, DunningRunnerInput{
			OrgId: "org_1", CampaignId: "dc_1", InitialFailureReason: "rate_limit",
		})

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())
		var res domain.DunningCampaign
		require.NoError(t, env.GetWorkflowResult(&res))
		assert.Equal(t, domain.DunningStatusRecovered, res.Status)
	})

	t.Run("exhausts all attempts and cancels subscription", func(t *testing.T) {
		var ts testsuite.WorkflowTestSuite
		env := ts.NewTestWorkflowEnvironment()
		da := &activities.DunningActivities{}
		env.RegisterActivity(da)

		config := domain.DunningConfig{
			ImmediateRetries: domain.ImmediateRetriesConfig{Enabled: false},
			ProgressiveRetries: domain.ProgressiveRetriesConfig{
				Enabled:     true,
				MaxAttempts: 1,
				Intervals:   []string{"1d"},
			},
		}

		env.OnActivity(da.LoadConfigForCampaign, mock.Anything, mock.Anything, mock.Anything).Return(config, nil)
		env.OnActivity(da.SendCommunication, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		env.OnWorkflow(DunningAttemptWorkflow, mock.Anything, mock.Anything).Return(domain.DunningAttempt{Status: domain.PaymentStatusFailed}, nil)
		env.OnActivity(da.UpdateCampaignWithAttemptResult, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(domain.DunningCampaign{Status: domain.DunningStatusActive}, nil)
		env.OnActivity(da.FailCampaignAndCancelSubscription, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(domain.DunningCampaign{Status: domain.DunningStatusFailed}, nil)

		env.ExecuteWorkflow(DunningRunnerWorkflow, DunningRunnerInput{OrgId: "org_1", CampaignId: "dc_1"})

		require.True(t, env.IsWorkflowCompleted())
		var res domain.DunningCampaign
		require.NoError(t, env.GetWorkflowResult(&res))
		assert.Equal(t, domain.DunningStatusFailed, res.Status)
		env.AssertCalled(t, "FailCampaignAndCancelSubscription", mock.Anything, "org_1", "dc_1", "all_attempts_failed")
	})

	t.Run("handles pause and resume", func(t *testing.T) {
		var ts testsuite.WorkflowTestSuite
		env := ts.NewTestWorkflowEnvironment()
		da := &activities.DunningActivities{}
		env.RegisterActivity(da)

		config := domain.DunningConfig{
			ProgressiveRetries: domain.ProgressiveRetriesConfig{
				Enabled:     true,
				MaxAttempts: 1,
				Intervals:   []string{"10d"},
			},
		}

		env.OnActivity(da.LoadConfigForCampaign, mock.Anything, mock.Anything, mock.Anything).Return(config, nil)

		// Signal pause after 1 day
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(SignalDunningPause, nil)
		}, 24*time.Hour)

		// Signal resume after another 2 days
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(SignalDunningResume, nil)
		}, 72*time.Hour)

		env.OnActivity(da.SendCommunication, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		env.OnWorkflow(DunningAttemptWorkflow, mock.Anything, mock.Anything).Return(domain.DunningAttempt{Status: domain.PaymentStatusSucceeded}, nil)
		env.OnActivity(da.UpdateCampaignWithAttemptResult, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(domain.DunningCampaign{Status: domain.DunningStatusRecovered}, nil)

		start := time.Now()
		env.ExecuteWorkflow(DunningRunnerWorkflow, DunningRunnerInput{OrgId: "org_1", CampaignId: "dc_1"})

		require.True(t, env.IsWorkflowCompleted())
		// Total wait should be 10d + the time it was paused (2 days extra between pause and resume signals)
		// Actually, awaitDunningInterval consumes the pause signal then returns dunningActionPaused.
		// Then it calls waitForResume which consumes the resume signal.
		// Then it continues to runDunningAttempt.
		// So total time = 1d (until pause) + 2d (until resume) = 3d approx?
		// No, the 10d timer is "restarted" or rather, the interval check was interrupted.
		// Looking at dunning_runner.go:
		// action := awaitDunningInterval(ctx, wait, ...)
		// if action == dunningActionPaused { waitForResume }
		// attempt, err := runDunningAttempt(...)
		// It doesn't resume the timer; it proceeds to the attempt immediately after resume.
		// So expected time is roughly 3 days.
		assert.WithinDuration(t, start.Add(72*time.Hour), env.Now(), 1*time.Minute)
	})
}
