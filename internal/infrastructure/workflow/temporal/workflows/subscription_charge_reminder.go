package workflows

import (
	"fmt"
	"payloop/internal/application/interfaces"
	"payloop/internal/domain/entities"
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

	return interfaces.Result{
		Success: true,
		Message: "sent",
		Payload: nil,
	}, nil
}
