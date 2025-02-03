package activities

import (
	"context"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/workflow"
)

type OrderActivities struct {
	orderService           services.OrderService
	subscriptionRepository repositories.SubscriptionRepository
	pubsub                 events.PubSub
}

func NewOrderActivities(orderService services.OrderService, subscriptionRepository repositories.SubscriptionRepository, pubsub events.PubSub) OrderActivities {
	return OrderActivities{
		orderService:           orderService,
		subscriptionRepository: subscriptionRepository,
		pubsub:                 pubsub,
	}
}

func (a *OrderActivities) CompleteOrder(ctx context.Context, data workflow.CompleteOrderStepInput) (workflow.Result, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CompleteOrder", "OrgId", data.PaymentContext.OrgId, "OrderId", data.PaymentContext.OrderId)

	order, err := a.orderService.CompleteOrder(ctx, orders.CompleteOrderCommand{
		OrgId:    data.PaymentContext.OrgId,
		OrderId:  data.PaymentContext.OrderId,
		Metadata: nil,
	})
	if err != nil {
		logger.Error("error completing order", "OrgId", data.PaymentContext.OrgId, "OrderId", data.PaymentContext.OrderId, "err", err.Error())
		return workflow.Result{}, temporal.NewNonRetryableApplicationError("Can't mark order as completed", "order", err)
	}

	// publish order completed event
	_ = a.pubsub.PublishJSON(events.TopicOrderCompleted, order)

	return workflow.Result{
		Success: true,
		Message: "Order completed",
		Payload: nil,
	}, nil
}

func (a *OrderActivities) GetOrderSubscriptions(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("GetOrderSubscriptions: ", "[OrgId]", orgId, "[OrderId]", orderId)
	// update the subscriptions
	subscriptions, err := a.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		logger.Error("Failed to find subscriptions", err.Error())
		return []entities.Subscription{}, err
	}

	return subscriptions, nil
}

func (a *OrderActivities) ChargeCustomerForBillingPeriod(ctx context.Context, customer entities.Customer, amount int) error {
	logger := activity.GetLogger(ctx)
	logger.Info("ChargeCustomerForBillingPeriod", "CustomerId", customer.ID, "Amount", amount)
	return nil
}
