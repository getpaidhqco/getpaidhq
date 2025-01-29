package activities

import (
	"context"
	"go.temporal.io/sdk/activity"
)

func CompleteOrderActivity(ctx context.Context, input interface{}) error {
	activity.GetLogger(ctx).Info("CompleteOrderActivity.", "input", input)

	return nil
}
