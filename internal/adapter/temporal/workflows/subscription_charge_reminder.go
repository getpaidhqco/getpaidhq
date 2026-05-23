package workflows

import (
	"time"

	temporalio "go.temporal.io/sdk/temporal"
	temporal "go.temporal.io/sdk/workflow"

	"getpaidhq/internal/adapter/temporal/activities"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// SubscriptionChargeReminder sleeps until the reminder time and then sends the
// renewal reminder. Mirrors
// internal/adapter/hatchet/workflows/subscription_charge_reminder.go.
func SubscriptionChargeReminder(ctx temporal.Context, input ReminderInput) (port.WorkflowResult, error) {
	logger := temporal.GetLogger(ctx)
	logger.Info("SubscriptionChargeReminder scheduled", "subscriptionId", input.Subscription.Id, "reminderAt", input.ReminderAt)

	wait := input.ReminderAt.Sub(temporal.Now(ctx))
	if wait > 0 {
		if err := temporal.Sleep(ctx, wait); err != nil {
			return port.WorkflowResult{}, err
		}
	}

	var act *activities.OrderActivities

	actCtx := temporal.WithActivityOptions(ctx, temporal.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporalio.RetryPolicy{
			InitialInterval:    time.Minute,
			BackoffCoefficient: 1.0,
			MaximumAttempts:    1,
		},
	})
	if err := temporal.ExecuteActivity(actCtx, act.ProcessReminderEvent, input.Subscription).
		Get(actCtx, nil); err != nil {
		return port.WorkflowResult{}, temporalio.NewNonRetryableApplicationError("SubscriptionChargeReminder failed", "reminder", err)
	}

	_ = domain.Subscription{}
	return port.WorkflowResult{Success: true, Message: "sent"}, nil
}
