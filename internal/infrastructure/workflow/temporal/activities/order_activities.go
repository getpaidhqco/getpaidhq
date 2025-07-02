package activities

import (
	"context"
	"encoding/json"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/settings"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/workflow/temporal/types"
	"payloop/internal/lib"
	"time"
)

type OrderActivities struct {
	orderService           interfaces.OrderWorkflowService
	subscriptionService    interfaces.SubscriptionService
	dunningService         interfaces.DunningService
	subscriptionRepository repositories.SubscriptionRepository
	settingRepository      repositories.SettingRepository
	paymentRepository      repositories.PaymentRepository
	pubsub                 events.PubSub
	gatewayFactory         factories.GatewayFactory
	errorReporter          lib.ErrorReporter
}

func NewOrderActivities(
	orderService interfaces.OrderWorkflowService,
	settingRepository repositories.SettingRepository,
	subscriptionService interfaces.SubscriptionService,
	dunningService interfaces.DunningService,
	subscriptionRepository repositories.SubscriptionRepository,
	pubsub events.PubSub,
	paymentRepository repositories.PaymentRepository,
	gatewayFactory factories.GatewayFactory,
	errorReporter lib.ErrorReporter,
) OrderActivities {
	return OrderActivities{
		gatewayFactory:         gatewayFactory,
		orderService:           orderService,
		subscriptionService:    subscriptionService,
		dunningService:         dunningService,
		subscriptionRepository: subscriptionRepository,
		paymentRepository:      paymentRepository,
		settingRepository:      settingRepository,
		pubsub:                 pubsub,
		errorReporter:          errorReporter,
	}
}

func (a *OrderActivities) CompleteOrder(ctx context.Context, paymentContext payment_providers.PaymentWebhookContext) (interfaces.Result, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CompleteCheckoutSession", "OrgId", paymentContext.OrgId, "OrderId", paymentContext.OrderId)

	order, err := a.orderService.CompleteCheckoutSession(ctx, orders.CompleteCheckoutSessionInput{
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

func (a *OrderActivities) HandlePaymentRefundedEvent(ctx context.Context, paymentContext payment_providers.PaymentWebhookContext) (interfaces.Result, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("HandlePaymentRefundedEvent", "OrgId", paymentContext.OrgId, "OrderId", paymentContext.OrderId)

	// Find the payment
	payment, err := a.paymentRepository.FindByPspId(ctx, paymentContext.OrgId, paymentContext.Payment.PspId)
	if err != nil {
		logger.Error("error finding payment", "OrgId", paymentContext.OrgId, "PspId", paymentContext.Payment.PspId, "err", err.Error())
		return interfaces.Result{}, temporal.NewNonRetryableApplicationError("can't find payment", "payment", err)
	}

	// update the payment status to refunded
	payment.Status = payments.PaymentStatusRefunded
	newPayment, err := a.paymentRepository.Update(ctx, payment)
	if err != nil {
		logger.Error("error completing order", "OrgId", paymentContext.OrgId, "OrderId", paymentContext.OrderId, "err", err.Error())
		return interfaces.Result{}, temporal.NewApplicationError("Can't update payment status", "payment", err)
	}

	// create the refund record
	_, err = a.paymentRepository.CreateRefund(ctx, entities.Refund{
		OrgId:      paymentContext.OrgId,
		Id:         lib.GenerateId("refund"),
		PaymentId:  payment.Id,
		Amount:     paymentContext.Payment.Amount,
		Currency:   paymentContext.Payment.Currency,
		Status:     entities.RefundStatusPending,
		RefundedAt: time.Now().UTC(), // paymentContext.Payment.PaidAt,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err != nil {
		logger.Error("error creating refund", "OrgId", paymentContext.OrgId, "PaymentId", payment.Id, "err", err.Error())
		return interfaces.Result{}, temporal.NewApplicationError("Can't create refund record", "refund", err)
	}

	return interfaces.Result{
		Success: true,
		Message: "Refund event processing",
		Payload: newPayment,
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
	logger.Info("ChargeCustomerForBillingPeriod", "orgId", currentSub.OrgId, "subscriptionId", currentSub.Id, "amount", currentSub.Amount)

	// Thin delegation to subscription service
	chargeResult, err := a.subscriptionService.ProcessSubscriptionCharge(ctx, currentSub)
	if err != nil {
		logger.Error("Failed to process subscription charge", "orgId", currentSub.OrgId, "subscriptionId", currentSub.Id, "error", err.Error())
		return payments.ChargeResult{}, err
	}

	logger.Info("Subscription charge completed", "orgId", currentSub.OrgId, "subscriptionId", currentSub.Id, "status", chargeResult.Status, "amount", chargeResult.Amount)
	return chargeResult, nil
}

func (a *OrderActivities) HandleChargeResult(ctx context.Context, subscription entities.Subscription, chargeResult payments.ChargeResult) (entities.Subscription, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("HandleChargeResult", "id", subscription.Id, "status", chargeResult.Status)

	if chargeResult.Status == payments.PaymentStatusSucceeded {
		return a.subscriptionService.HandleSubscriptionChargeSuccess(ctx, subscriptions.SubscriptionChargeInput{
			Subscription: subscription,
			ChargeResult: chargeResult,
		})
	} else {
		// Handle subscription charge failure
		updatedSubscription, err := a.subscriptionService.HandleSubscriptionChargeFailure(ctx, subscriptions.SubscriptionChargeInput{
			Subscription: subscription,
			ChargeResult: chargeResult,
		})
		if err != nil {
			logger.Error("Failed to handle subscription charge failure", "Error", err.Error())
			return subscription, err
		}
		return updatedSubscription, nil
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
func (a *OrderActivities) NotifyWorkflowEnded(ctx context.Context, orgId string, subId string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("NotifyWorkflowEnded", "OrgId", orgId, "SubscriptionId", subId)

	_ = a.pubsub.Publish(orgId, topic.SubscriptionWorkflowEnded, map[string]string{
		"orgId":           orgId,
		"subscription_id": subId,
	})
	return nil
}

func (a *OrderActivities) GetSubscription(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	return a.subscriptionRepository.FindById(ctx, orgId, id)
}

// GetSubscriptionSettings retrieves the subscription settings for the given organization.
// It returns a default settings.Subscription if the settings are not found or if an error occurs.
func (a *OrderActivities) GetSubscriptionSettings(ctx context.Context, orgId string) (settings.Subscription, error) {

	setting, err := a.subscriptionService.GetOrgSubscriptionSettings(ctx, orgId)
	if err != nil {
		logger := activity.GetLogger(ctx)
		logger.Warn("Failed to get Org Subscription settings", "OrgId", orgId, "error", err.Error())
		return settings.Subscription{}, nil
	}
	return setting, nil
}

func (a *OrderActivities) ProcessReminderEvent(ctx context.Context, subscription entities.Subscription) error {
	logger := activity.GetLogger(ctx)
	logger.Info("ProcessReminderEvent", "OrgId", subscription.OrgId, "SubscriptionId", subscription.Id)

	subSettings, _ := a.GetSubscriptionSettings(ctx, subscription.OrgId)

	if !subSettings.EmailReminders {
		logger.Info("Email reminders are disabled for this subscription, skipping reminder processing")
		return nil
	}

	subscription, err := a.subscriptionRepository.FindById(ctx, subscription.OrgId, subscription.Id)
	if err != nil {
		logger.Error("Failed to find subscription", "error", err.Error())
		return err
	}

	if subscription.Status != entities.SubscriptionStatusActive {
		logger.Info("Subscription is not active, skipping reminder processing", "Status", subscription.Status)
		return nil
	}

	err = a.pubsub.Publish(subscription.OrgId, topic.SubscriptionRenewalReminder, subscription)
	if err != nil {
		logger.Error("Failed to publish reminder event", "error", err.Error())
		return err
	}

	return nil
}
