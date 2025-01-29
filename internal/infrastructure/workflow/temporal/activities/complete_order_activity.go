package activities

import (
	"context"
	"go.temporal.io/sdk/activity"
)

func CompleteOrderActivity(ctx context.Context, expenseID WorkflowContext) error {
	activity.GetLogger(ctx).Info("Completing order.", "ExpenseID", expenseID)

	return nil
}
