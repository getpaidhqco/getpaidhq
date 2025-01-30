package activities

import (
	"context"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/workflow"
)

type OrderActivities struct {
	orderService services.OrderService
}

func NewOrderActivities(orderService services.OrderService) OrderActivities {
	return OrderActivities{
		orderService: orderService,
	}
}

func (a *OrderActivities) CompleteOrder(ctx context.Context, data workflow.CompleteOrderStepInput) (workflow.Result, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CompleteOrder", "OrgId", data.PaymentContext.OrgId, "OrderId", data.PaymentContext.OrderId)

	_, err := a.orderService.CompleteOrder(ctx, orders.CompleteOrderCommand{
		OrgId:    data.PaymentContext.OrgId,
		OrderId:  data.PaymentContext.OrderId,
		Metadata: nil,
	})
	if err != nil {
		logger.Error("error completing order", "OrgId", data.PaymentContext.OrgId, "OrderId", data.PaymentContext.OrderId, err.Error())
		return workflow.Result{}, temporal.NewNonRetryableApplicationError("Can't mark order as completed", "order", err)
	}

	return workflow.Result{
		Success: true,
		Message: "Order completed",
		Payload: nil,
	}, nil
}
