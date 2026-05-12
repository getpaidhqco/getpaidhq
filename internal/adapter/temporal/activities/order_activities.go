package activities

import (
	"context"
	"encoding/json"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"getpaidhq/internal/adapter/temporal/types"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// OrderActivities is the Temporal-side glue: every method is a thin wrapper
// around an engine-agnostic service. The wrappers exist only to translate
// service errors into Temporal-shaped retry decisions and to record activity
// logs; all business logic lives behind the port-level service interfaces.
type OrderActivities struct {
	orderService        port.OrderWorkflowService
	subscriptionService port.SubscriptionService
	paymentService      port.PaymentService
	subscriptionRepo    port.SubscriptionRepository
	settingRepository   port.SettingRepository
}

func NewOrderActivities(
	orderService port.OrderWorkflowService,
	subscriptionService port.SubscriptionService,
	paymentService port.PaymentService,
	subscriptionRepo port.SubscriptionRepository,
	settingRepository port.SettingRepository,
) OrderActivities {
	return OrderActivities{
		orderService:        orderService,
		subscriptionService: subscriptionService,
		paymentService:      paymentService,
		subscriptionRepo:    subscriptionRepo,
		settingRepository:   settingRepository,
	}
}

func (a *OrderActivities) CompleteOrder(ctx context.Context, paymentContext domain.PaymentWebhookContext) (port.WorkflowResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CompleteOrder", "OrgId", paymentContext.OrgId, "OrderId", paymentContext.OrderId)

	order, err := a.orderService.CompleteCheckoutSession(ctx, domain.CompleteCheckoutSessionInput{
		OrgId:          paymentContext.OrgId,
		OrderId:        paymentContext.OrderId,
		PaymentContext: paymentContext,
		Metadata:       nil,
	})
	if err != nil {
		return port.WorkflowResult{}, temporal.NewNonRetryableApplicationError("Can't mark order as completed", "order", err)
	}

	return port.WorkflowResult{Success: true, Message: "Order completed", Payload: order}, nil
}

func (a *OrderActivities) HandlePaymentRefundedEvent(ctx context.Context, paymentContext domain.PaymentWebhookContext) (port.WorkflowResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("HandlePaymentRefundedEvent", "OrgId", paymentContext.OrgId, "OrderId", paymentContext.OrderId)

	payment, err := a.paymentService.ProcessRefund(ctx, paymentContext)
	if err != nil {
		return port.WorkflowResult{}, temporal.NewApplicationError("Can't process refund", "refund", err)
	}

	return port.WorkflowResult{Success: true, Message: "Refund event processing", Payload: payment}, nil
}

func (a *OrderActivities) GetOrderSubscriptions(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("GetOrderSubscriptions: ", "[OrgId]", orgId, "[OrderId]", orderId)

	return a.subscriptionRepo.FindByOrderId(ctx, orgId, orderId)
}

// ChargeCustomerForBillingPeriod delegates to SubscriptionService and rewraps
// gateway-side failures as retryable Temporal application errors so Temporal's
// retry policy can kick in.
func (a *OrderActivities) ChargeCustomerForBillingPeriod(ctx context.Context, currentSub domain.Subscription) (domain.ChargeResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ChargeCustomerForBillingPeriod", "id", currentSub.Id, "Total", currentSub.Amount)

	result, err := a.subscriptionService.ChargeForBillingPeriod(ctx, currentSub)
	if err != nil {
		return domain.ChargeResult{}, temporal.NewApplicationError(err.Error(), "gateway_error", nil)
	}
	return result, nil
}

func (a *OrderActivities) HandleChargeResult(ctx context.Context, subscription domain.Subscription, chargeResult domain.ChargeResult) (domain.Subscription, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("HandleChargeResult", "id", subscription.Id)

	if chargeResult.Status == domain.PaymentStatusSucceeded {
		return a.subscriptionService.HandleSubscriptionChargeSuccess(ctx, domain.SubscriptionChargeInput{
			Subscription: subscription,
			ChargeResult: chargeResult,
		})
	}
	return a.subscriptionService.HandleSubscriptionChargeFailure(ctx, domain.SubscriptionChargeInput{
		Subscription: subscription,
		ChargeResult: chargeResult,
	})
}

// StoreSubscriptionWorkflowContext stores the Temporal workflow Id and workflow run Id
// so that the system can query the workflow status later.
//
// This is Temporal-specific glue (the Execution type is Temporal's) and stays
// in the adapter rather than moving down to a service.
func (a *OrderActivities) StoreSubscriptionWorkflowContext(ctx context.Context, input types.StoreSubscriptionWorkflowContextInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("StoreSubscriptionWorkflowContext", "OrgId", input.OrgId, "SubscriptionId", input.SubscriptionId, "Execution", input.Execution)
	executionBytes, err := json.Marshal(input.Execution)
	if err != nil {
		return err
	}

	_, err = a.settingRepository.Create(ctx, domain.Setting{
		OrgId:    input.OrgId,
		ParentId: input.SubscriptionId,
		Id:       "temporal-workflow",
		Type:     "workflow.Execution",
		Value:    string(executionBytes),
	})
	return err
}

func (a *OrderActivities) ErrorState(ctx context.Context, subscription domain.Subscription, err error) error {
	logger := activity.GetLogger(ctx)
	logger.Info("ErrorState", "OrgId", subscription.OrgId, "SubscriptionId", subscription.Id, "err", err.Error())
	return a.subscriptionService.MarkAsError(ctx, subscription, err)
}

func (a *OrderActivities) GetSubscription(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	return a.subscriptionRepo.FindById(ctx, orgId, id)
}

func (a *OrderActivities) ProcessReminderEvent(ctx context.Context, subscription domain.Subscription) error {
	logger := activity.GetLogger(ctx)
	logger.Info("ProcessReminderEvent", "OrgId", subscription.OrgId, "SubscriptionId", subscription.Id)
	return a.subscriptionService.SendRenewalReminder(ctx, subscription.OrgId, subscription.Id)
}
