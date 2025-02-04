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
	"payloop/internal/lib"
	"time"
)

type OrderActivities struct {
	orderService           services.OrderService
	subscriptionRepository repositories.SubscriptionRepository
	paymentRepository      repositories.PaymentRepository
	pubsub                 events.PubSub
}

func NewOrderActivities(orderService services.OrderService, subscriptionRepository repositories.SubscriptionRepository, pubsub events.PubSub, paymentRepository repositories.PaymentRepository) OrderActivities {
	return OrderActivities{
		orderService:           orderService,
		subscriptionRepository: subscriptionRepository,
		paymentRepository:      paymentRepository,
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

// ChargeCustomerForBillingPeriod is responsible for charging the customer for the billing period and to
// update the subscription status to reflect the billing period
// TODO move this to the subscription service
// TODO split the charge and DB work into 2 activities
func (a *OrderActivities) ChargeCustomerForBillingPeriod(ctx context.Context, subscription entities.Subscription) (entities.Subscription, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ChargeCustomerForBillingPeriod", "id", subscription.Id, "Amount", subscription.Amount)

	// TODO success charge

	// create a payment
	matadata := make(map[string]string)
	matadata["psp_id"] = "mocky"
	payment := entities.Payment{
		OrgId:          subscription.OrgId,
		Id:             lib.GenerateId("pmt"),
		OrderId:        subscription.OrderId,
		SubscriptionId: subscription.Id,
		Status:         entities.PaymentStatusSucceeded,
		Currency:       subscription.Currency,
		Amount:         subscription.Amount,
		PspFee:         0,
		PlatformFee:    0,
		NetAmount:      subscription.Amount,
		Metadata:       matadata,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	payment, err := a.paymentRepository.Create(ctx, payment)
	if err != nil {
		logger.Error("Failed to create payment", err.Error())
	}

	// update the subscription status
	lastCharge := time.Now().UTC()
	subscription.Status = entities.SubscriptionStatusActive
	subscription.CyclesProcessed++
	subscription.TotalRevenue += subscription.Amount
	subscription.LastCharge = &lastCharge

	nextCharge := subscription.NextBillingDate()
	subscription.RenewsAt = &nextCharge

	logger.Info("Subscription charged, updating with new values",
		"id", subscription.Id,
		"NextCharge", nextCharge,
		"cycles", subscription.CyclesProcessed)
	newSub, err := a.subscriptionRepository.Update(ctx, subscription)
	return newSub, err
}

func (a *OrderActivities) GetSubscription(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	return a.subscriptionRepository.FindById(ctx, orgId, id)
}
