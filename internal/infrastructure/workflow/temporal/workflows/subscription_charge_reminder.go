package workflows

import (
	"fmt"
	temporalio "go.temporal.io/sdk/temporal"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"time"

	temporal_wf "go.temporal.io/sdk/workflow"
)

func SubscriptionChargeReminder(ctx temporal_wf.Context, subscription entities.Subscription, reminderTime time.Time) (interfaces.Result, error) {
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
		return interfaces.Result{Success: false}, err
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
		return interfaces.Result{
			Success: false,
		}, temporalio.NewNonRetryableApplicationError("SubscriptionChargeReminder failed", "", err)
	}

	return interfaces.Result{
		Success: true,
		Message: "sent",
		Payload: nil,
	}, nil
}
