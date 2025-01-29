package activities

import (
	"context"
	"go.temporal.io/sdk/activity"
	"payloop/internal/domain/workflow"
)

func CompleteOrder(ctx context.Context, payload workflow.CompleteOrderStepInput) (workflow.Result, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CompleteOrder", "OrgId", payload.OrgId, "OrderId", payload.OrderId)
	return workflow.Result{}, nil
}
