package workflows

import (
	"fmt"
	temporalio "go.temporal.io/sdk/temporal"
	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"time"

	temporal_wf "go.temporal.io/sdk/workflow"
)

func SubscriptionChargeReminder(ctx temporal_wf.Context, subscription domain.Subscription, reminderTime time.Time) (port.WorkflowResult, error) {
	logger := temporal_wf.GetLogger(ctx)
	logger.Info("SubscriptionChargeReminder started, waiting for reminder time", "reminderTime", reminderTime)

	valid := true

	duration := reminderTime.Sub(temporal_wf.Now(ctx))
	ok, err := temporal_wf.AwaitWithTimeout(ctx, duration, func() bool {
		rollover := temporal_wf.GetInfo(ctx).GetContinueAsNewSuggested()
		return !valid || rollover
	})
	if err != nil {
		logger.Error("Reminder interrupted, not processing", "Error", err)
		return port.WorkflowResult{Success: false}, err
	}
	if !ok {
		logger.Info(fmt.Sprintf("REMINDER EMAIL FOR [%s][%s]", subscription.OrgId, subscription.Id))
	}

	var a *activities.OrderActivities
	// ACTIVITY
	// Complete the Order
	ctx1 := temporal_wf.WithActivityOptions(ctx, temporal_wf.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
			MaximumAttempts:    1,
		},
	})
	err = temporal_wf.ExecuteActivity(ctx1, a.ProcessReminderEvent, subscription).
		Get(ctx1, nil)
	if err != nil {
		logger.Error("[SubscriptionChargeReminder] failed with error: ", "Error", err.Error())
		return port.WorkflowResult{
			Success: false,
		}, temporalio.NewNonRetryableApplicationError("SubscriptionChargeReminder failed", "", err)
	}

	return port.WorkflowResult{
		Success: true,
		Message: "sent",
		Payload: nil,
	}, nil
}
