package activities

import (
	"context"
	"encoding/json"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// OrderActivities is the Temporal-side glue: every method is a thin wrapper
// around an engine-agnostic service. The wrappers exist only to translate
// service errors into Temporal-shaped retry decisions and to record activity
// logs; all business logic lives behind the port-level service interfaces.
//
// This is the Temporal mirror of internal/adapter/hatchet/steps/*.go.
type OrderActivities struct {
	orderService        port.OrderWorkflowService
	subscriptionService port.SubscriptionService
	paymentService      port.PaymentService
	subscriptionRepo    port.SubscriptionRepository
	reminderResolver    port.ReminderConfigResolver
}

func NewOrderActivities(
	orderService port.OrderWorkflowService,
	subscriptionService port.SubscriptionService,
	paymentService port.PaymentService,
	subscriptionRepo port.SubscriptionRepository,
	reminderResolver port.ReminderConfigResolver,
) OrderActivities {
	return OrderActivities{
		orderService:        orderService,
		subscriptionService: subscriptionService,
		paymentService:      paymentService,
		subscriptionRepo:    subscriptionRepo,
		reminderResolver:    reminderResolver,
	}
}

func (a *OrderActivities) log(ctx context.Context, msg string, keyvals ...any) {
	defer func() { recover() }()
	activity.GetLogger(ctx).Info(msg, keyvals...)
}

func (a *OrderActivities) CompleteOrder(ctx context.Context, paymentContext domain.PaymentWebhookContext) (domain.Order, error) {
	a.log(ctx, "CompleteOrder", "OrgId", paymentContext.OrgId, "OrderId", paymentContext.OrderId)

	order, err := a.orderService.CompleteCheckoutSession(ctx, domain.CompleteCheckoutSessionInput{
		OrgId:          paymentContext.OrgId,
		OrderId:        paymentContext.OrderId,
		PaymentContext: paymentContext,
		Metadata:       nil,
	})
	if err != nil {
		return domain.Order{}, temporal.NewNonRetryableApplicationError("Can't mark order as completed", "order", err)
	}
	return order, nil
}

func (a *OrderActivities) HandlePaymentRefundedEvent(ctx context.Context, paymentContext domain.PaymentWebhookContext) (domain.Payment, error) {
	a.log(ctx, "HandlePaymentRefundedEvent", "OrgId", paymentContext.OrgId, "OrderId", paymentContext.OrderId)

	payment, err := a.paymentService.ProcessRefund(ctx, paymentContext)
	if err != nil {
		return domain.Payment{}, temporal.NewApplicationError("Can't process refund", "refund", err)
	}
	return payment, nil
}

func (a *OrderActivities) GetOrderSubscriptions(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
	a.log(ctx, "GetOrderSubscriptions", "OrgId", orgId, "OrderId", orderId)
	return a.subscriptionRepo.FindByOrderId(ctx, orgId, orderId)
}

// ChargeCustomerForBillingPeriod delegates to SubscriptionService and rewraps
// gateway-side failures as retryable Temporal application errors so Temporal's
// retry policy can kick in.
func (a *OrderActivities) ChargeCustomerForBillingPeriod(ctx context.Context, currentSub domain.Subscription) (domain.ChargeResult, error) {
	a.log(ctx, "ChargeCustomerForBillingPeriod", "id", currentSub.Id, "Total", currentSub.Amount)

	result, err := a.subscriptionService.ChargeForBillingPeriod(ctx, currentSub)
	if err != nil {
		return domain.ChargeResult{}, temporal.NewApplicationError(err.Error(), "gateway_error", nil)
	}
	return result, nil
}

func (a *OrderActivities) HandleChargeResult(ctx context.Context, subscription domain.Subscription, chargeResult domain.ChargeResult) (domain.Subscription, error) {
	a.log(ctx, "HandleChargeResult", "id", subscription.Id)

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

func (a *OrderActivities) ErrorState(ctx context.Context, subscription domain.Subscription, errMsg string) error {
	a.log(ctx, "ErrorState", "OrgId", subscription.OrgId, "SubscriptionId", subscription.Id, "err", errMsg)
	return a.subscriptionService.MarkAsError(ctx, subscription, &activityErr{errMsg})
}

func (a *OrderActivities) GetSubscription(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	return a.subscriptionRepo.FindById(ctx, orgId, id)
}

// ResolveReminderConfig exposes the per-tenant reminder policy to the durable
// SubscriptionWorkflow. Workflows can't do I/O directly, so this activity wraps
// the shared resolver — the same one the Hatchet sweep uses.
func (a *OrderActivities) ResolveReminderConfig(ctx context.Context, orgId string) (domain.ReminderConfig, error) {
	return a.reminderResolver.ResolveReminderConfig(ctx, orgId)
}

func (a *OrderActivities) ProcessReminderEvent(ctx context.Context, subscription domain.Subscription) error {
	a.log(ctx, "ProcessReminderEvent", "OrgId", subscription.OrgId, "SubscriptionId", subscription.Id)
	return a.subscriptionService.SendRenewalReminder(ctx, subscription.OrgId, subscription.Id)
}

// activityErr is a stand-in error type for ErrorState — Temporal activities
// cannot carry a Go error across the boundary, so the caller passes a string
// and we wrap it here for MarkAsError.
type activityErr struct{ msg string }

func (e *activityErr) Error() string { return e.msg }

var _ = json.Marshal
