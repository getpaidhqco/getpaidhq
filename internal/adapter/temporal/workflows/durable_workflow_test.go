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
)

// ---- SubscriptionWorkflow: the per-subscription long-running runner ----

func TestSubscriptionWorkflow_TerminalStatusExitsImmediately(t *testing.T) {
	cases := []struct {
		name   string
		status domain.SubscriptionStatus
	}{
		{"cancelled", domain.SubscriptionStatusCancelled},
		{"expired", domain.SubscriptionStatusExpired},
		{"completed", domain.SubscriptionStatusCompleted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ts testsuite.WorkflowTestSuite
			env := ts.NewTestWorkflowEnvironment()

			env.ExecuteWorkflow(SubscriptionWorkflow, domain.Subscription{
				OrgId: "org_1", Id: "sub_1", Status: tc.status,
			})

			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError(), "terminal-status workflow returns clean")
		})
	}
}

func TestSubscriptionWorkflow_NoNextChargeDateExits(t *testing.T) {
	// Active status but no billing interval → GetNextChargeDate is zero → break.
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	env.ExecuteWorkflow(SubscriptionWorkflow, domain.Subscription{
		OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestSubscriptionWorkflow_CancelSignalBreaksTheLoop(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	// Reminder and billing children are registered as stubs so the workflow
	// can spawn them without running their real bodies. The cancel signal
	// fires while the main loop is awaiting the next charge time. The runner
	// also resolves the reminder config once per cycle via an activity.
	oa := &activities.OrderActivities{}
	env.RegisterActivity(oa)
	env.OnActivity(oa.ResolveReminderConfig, mock.Anything, mock.Anything).
		Return(domain.ReminderConfig{}, nil).Maybe()
	env.RegisterWorkflow(SubscriptionChargeReminder)
	env.RegisterWorkflow(BillingCycleWorkflow)
	env.OnWorkflow(SubscriptionChargeReminder, mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	env.OnWorkflow(BillingCycleWorkflow, mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalCancelRunner, domain.Subscription{})
	}, 30*time.Second)

	env.ExecuteWorkflow(SubscriptionWorkflow, domain.Subscription{
		OrgId:              "org_1",
		Id:                 "sub_1",
		Status:             domain.SubscriptionStatusActive,
		BillingInterval:    domain.BillingInterval("month"),
		BillingIntervalQty: 1,
		RenewsAt:           env.Now().Add(24 * time.Hour),
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestSubscriptionWorkflow_FullCycleCompletesOnLastBill(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	oa := &activities.OrderActivities{}
	env.RegisterActivity(oa)
	env.RegisterWorkflow(SubscriptionChargeReminder)
	env.RegisterWorkflow(BillingCycleWorkflow)

	// One reminder child fires (we don't care about its return), then the
	// billing child returns a successful ChargeResult, then HandleChargeResult
	// returns a subscription marked Completed so the runner breaks out.
	env.OnActivity(oa.ResolveReminderConfig, mock.Anything, mock.Anything).
		Return(domain.ReminderConfig{Enabled: true, Offsets: []time.Duration{30 * time.Minute}}, nil).Maybe()
	env.OnWorkflow(SubscriptionChargeReminder, mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	env.OnWorkflow(BillingCycleWorkflow, mock.Anything, mock.Anything).
		Return(domain.ChargeResult{Status: domain.PaymentStatusSucceeded, Amount: 1000}, nil)
	env.OnActivity(oa.HandleChargeResult, mock.Anything, mock.Anything, mock.Anything).
		Return(domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusCompleted}, nil)

	env.ExecuteWorkflow(SubscriptionWorkflow, domain.Subscription{
		OrgId:              "org_1",
		Id:                 "sub_1",
		Status:             domain.SubscriptionStatusActive,
		BillingInterval:    domain.BillingInterval("month"),
		BillingIntervalQty: 1,
		Cycles:             1,
		RenewsAt:           env.Now().Add(time.Hour),
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var got domain.Subscription
	require.NoError(t, env.GetWorkflowResult(&got))
	assert.Equal(t, domain.SubscriptionStatusCompleted, got.Status, "completed status persisted through the loop")
}

// ---- DunningRunnerWorkflow ----

func TestDunningRunnerWorkflow_ConfigLoadFailureSurfacesAsWorkflowError(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()

	da := &activities.DunningActivities{}
	env.RegisterActivity(da)
	env.OnActivity(da.LoadConfigForCampaign, mock.Anything, mock.Anything, mock.Anything).
		Return(domain.DunningConfig{}, errors.New("config repo down"))

	env.ExecuteWorkflow(DunningRunnerWorkflow, DunningRunnerInput{
		OrgId: "org_1", CampaignId: "dc_1", SubscriptionId: "sub_1", FailedAmount: 5000, Currency: "USD",
	})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError(), "config load failure aborts the runner")
}
