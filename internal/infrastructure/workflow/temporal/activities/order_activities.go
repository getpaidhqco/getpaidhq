package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"math/rand"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/workflow/temporal/types"
)

type OrderActivities struct {
	orderService           interfaces.OrderService
	subscriptionService    interfaces.SubscriptionService
	subscriptionRepository repositories.SubscriptionRepository
	settingRepository      repositories.SettingRepository
	paymentRepository      repositories.PaymentRepository
	pubsub                 events.PubSub
	gatewayFactory         factories.GatewayFactory
}

func NewOrderActivities(
	orderService interfaces.OrderService,
	settingRepository repositories.SettingRepository,
	subscriptionService interfaces.SubscriptionService,
	subscriptionRepository repositories.SubscriptionRepository,
	pubsub events.PubSub,
	paymentRepository repositories.PaymentRepository,
	gatewayFactory factories.GatewayFactory,
) OrderActivities {
	return OrderActivities{
		gatewayFactory:         gatewayFactory,
		orderService:           orderService,
		subscriptionService:    subscriptionService,
		subscriptionRepository: subscriptionRepository,
		paymentRepository:      paymentRepository,
		settingRepository:      settingRepository,
		pubsub:                 pubsub,
	}
}

func (a *OrderActivities) CompleteOrder(ctx context.Context, paymentContext payment_providers.PaymentWebhookContext) (interfaces.Result, error) {
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
		return interfaces.Result{}, temporal.NewNonRetryableApplicationError("Can't mark order as completed", "order", err)
	}

	return interfaces.Result{
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
func (a *OrderActivities) ChargeCustomerForBillingPeriod(ctx context.Context, currentSub entities.Subscription) (payments.ChargeResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ChargeCustomerForBillingPeriod", "id", currentSub.Id, "Amount", currentSub.Amount)

	subscription, err := a.subscriptionRepository.FindById(ctx, currentSub.OrgId, currentSub.Id)
	if err != nil {
		logger.Error("Failed to find subscription", "error", err.Error())
		return payments.ChargeResult{}, err
	}

	gw, err := a.gatewayFactory.NewGateway(ctx, subscription.OrgId, subscription.PspId)
	if err != nil {
		logger.Error("Failed to get gateway", err.Error())
		return payments.ChargeResult{}, err
	}

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

	randomNumber := rand.Intn(101) // Generate a random number between 0 and 100
	fmt.Println(randomNumber)

	chargeResult := gw.ChargePayment(ctx, payment_providers.ChargePaymentCommand{
		OrgId:    subscription.OrgId,
		Amount:   subscription.Amount,
		Currency: subscription.Currency,
		//Reference: fmt.Sprintf("%s_%d_%d", subscription.Id, subscription.CyclesProcessed+1, randomNumber),
		PaymentMethod: payment_providers.PaymentMethod{
			PspId:       paymentMethod.Id,
			Name:        paymentMethod.Name,
			Type:        paymentMethod.Type,
			IsRecurring: true,
			Token:       paymentMethod.Token,
		},
		Customer: customer,
	})
	if !chargeResult.Success && !chargeResult.Retryable {
		return payments.ChargeResult{}, errors.New("failed to charge customer")
	}
	rawData, err := json.Marshal(chargeResult.PspResponse)
	if err != nil {
		logger.Error("failed to marshal charge result", "error", err.Error())
	}

	if chargeResult.Success {
		result := payments.ChargeResult{
			Amount:    chargeResult.AmountCharged,
			Status:    payments.PaymentStatusSucceeded,
			Currency:  subscription.Currency,
			PspId:     chargeResult.PspId,
			Reference: chargeResult.Reference,
			RawData:   string(rawData),
		}
		return result, nil
	} else {
		result := payments.ChargeResult{
			Amount:   0,
			Status:   payments.PaymentStatusFailed,
			Currency: subscription.Currency,
			PspId:    chargeResult.PspId,
			RawData:  string(rawData),
		}
		return result, nil
	}

}

func (a *OrderActivities) HandleChargeResult(ctx context.Context, subscription entities.Subscription, chargeResult payments.ChargeResult) (entities.Subscription, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("HandleChargeResult", "id", subscription.Id)

	if chargeResult.Status == payments.PaymentStatusSucceeded {
		return a.subscriptionService.HandleSubscriptionChargeSuccess(ctx, subscriptions.SubscriptionChargeInput{
			Subscription: subscription,
			ChargeResult: chargeResult,
		})
	} else {
		return a.subscriptionService.HandleSubscriptionChargeFailure(ctx, subscriptions.SubscriptionChargeInput{
			Subscription: subscription,
			ChargeResult: chargeResult,
		})
	}
}

// StoreSubscriptionWorkflowContext stores the Temporal workflow Id and workflow run Id
// so that the system can query the workflow status later.
//
// At the moment this is not an Application level concern, only a Temporal concern, so use the
// repositories directly here instead of a Service implementation.
func (a *OrderActivities) StoreSubscriptionWorkflowContext(ctx context.Context, input types.StoreSubscriptionWorkflowContextInput) error {
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

func (a *OrderActivities) ErrorState(ctx context.Context, subscription entities.Subscription, err error) error {
	logger := activity.GetLogger(ctx)
	logger.Info("ErrorState", "OrgId", subscription.OrgId, "SubscriptionId", subscription.Id, "err", err.Error())

	subscription.Status = entities.SubscriptionStatusError
	subscription.Metadata["error"] = err.Error()

	_, err = a.subscriptionRepository.Update(ctx, subscription)
	if err != nil {
		logger.Error("Failed to update subscription", "error", err.Error())
		return err
	}

	return nil
}

func (a *OrderActivities) GetSubscription(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	return a.subscriptionRepository.FindById(ctx, orgId, id)
}
