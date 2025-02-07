package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/workflow"

	temporal_workflow "go.temporal.io/sdk/workflow"
)

type StoreSubscriptionWorkflowContextInput struct {
	OrgId          string
	SubscriptionId string
	Execution      temporal_workflow.Execution
}

type OrderActivities struct {
	orderService           services.OrderService
	subscriptionService    services.SubscriptionService
	subscriptionRepository repositories.SubscriptionRepository
	settingRepository      repositories.SettingRepository
	paymentRepository      repositories.PaymentRepository
	pubsub                 events.PubSub
	paymentGateway         payment_providers.Gateway
}

func NewOrderActivities(
	orderService services.OrderService,
	settingRepository repositories.SettingRepository,
	subscriptionService services.SubscriptionService,
	subscriptionRepository repositories.SubscriptionRepository,
	pubsub events.PubSub,
	paymentRepository repositories.PaymentRepository,
	paymentGateway payment_providers.Gateway,
) OrderActivities {
	return OrderActivities{
		paymentGateway:         paymentGateway,
		orderService:           orderService,
		subscriptionService:    subscriptionService,
		subscriptionRepository: subscriptionRepository,
		paymentRepository:      paymentRepository,
		settingRepository:      settingRepository,
		pubsub:                 pubsub,
	}
}

func (a *OrderActivities) CompleteOrder(ctx context.Context, paymentContext payment_providers.PaymentWebhookContext) (workflow.Result, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CompleteOrder", "OrgId", paymentContext.OrgId, "OrderId", paymentContext.OrderId)

	order, err := a.orderService.CompleteOrder(ctx, orders.CompleteOrderCommand{
		OrgId:          paymentContext.OrgId,
		OrderId:        paymentContext.OrderId,
		PaymentContext: paymentContext,
		Metadata:       nil,
	})
	if err != nil {
		logger.Error("error completing order", "OrgId", paymentContext.OrgId, "OrderId", paymentContext.OrderId, "err", err.Error())
		return workflow.Result{}, temporal.NewNonRetryableApplicationError("Can't mark order as completed", "order", err)
	}

	return workflow.Result{
		Success: true,
		Message: "Order completed",
		Payload: order,
	}, nil
}

func (a *OrderActivities) GetOrderSubscriptions(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("GetOrderSubscriptions: ", "[OrgId]", orgId, "[OrderId]", orderId)

	subs, err := a.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)

	return subs, err
}

// ChargeCustomerForBillingPeriod is responsible for charging the customer for the billing period and to
// update the subscription status to reflect the billing period
func (a *OrderActivities) ChargeCustomerForBillingPeriod(ctx context.Context, subscription entities.Subscription) (payments.ChargeResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ChargeCustomerForBillingPeriod", "id", subscription.Id, "Amount", subscription.Amount)

	customer, err := a.subscriptionService.GetSubscriptionCustomer(ctx, subscription)
	if err != nil {
		logger.Error("failed to get customer", "error", err.Error())
		return payments.ChargeResult{}, err
	}

	paymentMethod, err := a.subscriptionService.GetSubscriptionPaymentMethod(ctx, subscription)
	if err != nil {
		logger.Error("failed to get paymentMethod", "error", err.Error())
		return payments.ChargeResult{}, err
	}

	chargeResult, err := a.paymentGateway.ChargePayment(ctx, payment_providers.ChargePaymentCommand{
		OrgId:     subscription.OrgId,
		Amount:    subscription.Amount,
		Currency:  subscription.Currency,
		Reference: fmt.Sprintf("%s_%d", subscription.Id, subscription.CyclesProcessed+1),
		PaymentMethod: payment_providers.PaymentMethod{
			PspId:       paymentMethod.Id,
			Name:        paymentMethod.Name,
			Type:        paymentMethod.Type,
			IsRecurring: true,
			Token:       paymentMethod.Token,
		},
		Customer: customer,
	})
	if err != nil {
		return payments.ChargeResult{}, err
	}
	rawData, err := json.Marshal(chargeResult.PspResponse)
	if err != nil {
		logger.Error("failed to marshal charge result", "error", err.Error())
	}
	result := payments.ChargeResult{
		Amount:   chargeResult.AmountCharged,
		Status:   payments.PaymentStatusSucceeded,
		Currency: subscription.Currency,
		PspId:    chargeResult.PspId,
		RawData:  string(rawData),
	}
	return result, nil
}

func (a *OrderActivities) StoreChargeResults(ctx context.Context, subscription entities.Subscription, chargeResult payments.ChargeResult) (entities.Subscription, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("StoreChargeResults", "id", subscription.Id)

	newSub, err := a.subscriptionService.HandleSubscriptionChargeSuccess(ctx, subscriptions.SubscriptionChargeSuccessInput{
		Subscription: subscription,
		ChargeResult: chargeResult,
	})
	return newSub, err
}

// StoreSubscriptionWorkflowContext stores the Temporal workflow Id and workflow run Id
// so that the system can query the workflow status later.
//
// At the moment this is not an Application level concern, only a Temporal concern, so use the
// repositories directly here instead of a Service implementation.
func (a *OrderActivities) StoreSubscriptionWorkflowContext(ctx context.Context, input StoreSubscriptionWorkflowContextInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("StoreSubscriptionWorkflowContext", "OrgId", input.OrgId, "SubscriptionId", input.SubscriptionId, "Execution", input.Execution)
	executionBytes, err := json.Marshal(input.Execution)
	if err != nil {
		return err
	}

	_, err = a.settingRepository.Create(ctx, entities.Setting{
		OrgId:    input.OrgId,
		ParentId: input.SubscriptionId,
		Id:       "temporal-workflow",
		Type:     "workflow.Execution",
		Value:    string(executionBytes),
	})

	return err
}

func (a *OrderActivities) GetSubscription(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	return a.subscriptionRepository.FindById(ctx, orgId, id)
}
