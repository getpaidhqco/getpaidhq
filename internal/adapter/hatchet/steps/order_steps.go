package steps

import (
	"context"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

// OrderSteps is the Hatchet-side glue for order/subscription business logic.
// Each method is a thin wrapper around an engine-agnostic narrow service so
// the workflows can remain free of business rules.
type OrderSteps struct {
	logger              port.Logger
	orderService        port.OrderWorkflowService
	subscriptionService port.SubscriptionService
	paymentService      port.PaymentService
	subscriptionRepo    port.SubscriptionRepository
	settingRepository   port.SettingRepository
}

func NewOrderSteps(
	logger port.Logger,
	orderService port.OrderWorkflowService,
	subscriptionService port.SubscriptionService,
	paymentService port.PaymentService,
	subscriptionRepo port.SubscriptionRepository,
	settingRepository port.SettingRepository,
) *OrderSteps {
	return &OrderSteps{
		logger:              logger,
		orderService:        orderService,
		subscriptionService: subscriptionService,
		paymentService:      paymentService,
		subscriptionRepo:    subscriptionRepo,
		settingRepository:   settingRepository,
	}
}

func (s *OrderSteps) CompleteOrder(ctx context.Context, pc domain.PaymentWebhookContext) (port.WorkflowResult, error) {
	s.logger.Info("CompleteOrder", "OrgId", pc.OrgId, "OrderId", pc.OrderId)
	order, err := s.orderService.CompleteCheckoutSession(ctx, domain.CompleteCheckoutSessionInput{
		OrgId:          pc.OrgId,
		OrderId:        pc.OrderId,
		PaymentContext: pc,
		Metadata:       nil,
	})
	if err != nil {
		return port.WorkflowResult{}, err
	}
	return port.WorkflowResult{Success: true, Message: "Order completed", Payload: order}, nil
}

func (s *OrderSteps) HandlePaymentRefundedEvent(ctx context.Context, pc domain.PaymentWebhookContext) (port.WorkflowResult, error) {
	s.logger.Info("HandlePaymentRefundedEvent", "OrgId", pc.OrgId, "OrderId", pc.OrderId)
	payment, err := s.paymentService.ProcessRefund(ctx, pc)
	if err != nil {
		return port.WorkflowResult{}, err
	}
	return port.WorkflowResult{Success: true, Message: "Refund event processing", Payload: payment}, nil
}

func (s *OrderSteps) GetOrderSubscriptions(ctx context.Context, orgId, orderId string) ([]domain.Subscription, error) {
	s.logger.Info("GetOrderSubscriptions", "OrgId", orgId, "OrderId", orderId)
	return s.subscriptionRepo.FindByOrderId(ctx, orgId, orderId)
}

func (s *OrderSteps) ChargeCustomerForBillingPeriod(ctx context.Context, sub domain.Subscription) (domain.ChargeResult, error) {
	s.logger.Info("ChargeCustomerForBillingPeriod", "id", sub.Id, "amount", sub.Amount)
	return s.subscriptionService.ChargeForBillingPeriod(ctx, sub)
}

func (s *OrderSteps) HandleChargeResult(ctx context.Context, sub domain.Subscription, result domain.ChargeResult) (domain.Subscription, error) {
	s.logger.Info("HandleChargeResult", "id", sub.Id, "status", result.Status)
	if result.Status == domain.PaymentStatusSucceeded {
		return s.subscriptionService.HandleSubscriptionChargeSuccess(ctx, domain.SubscriptionChargeInput{
			Subscription: sub,
			ChargeResult: result,
		})
	}
	return s.subscriptionService.HandleSubscriptionChargeFailure(ctx, domain.SubscriptionChargeInput{
		Subscription: sub,
		ChargeResult: result,
	})
}

func (s *OrderSteps) ErrorState(ctx context.Context, sub domain.Subscription, runErr error) error {
	s.logger.Info("ErrorState", "OrgId", sub.OrgId, "SubscriptionId", sub.Id, "err", runErr.Error())
	return s.subscriptionService.MarkAsError(ctx, sub, runErr)
}

func (s *OrderSteps) GetSubscription(ctx context.Context, orgId, id string) (domain.Subscription, error) {
	return s.subscriptionRepo.FindById(ctx, orgId, id)
}

func (s *OrderSteps) ProcessReminderEvent(ctx context.Context, sub domain.Subscription) error {
	s.logger.Info("ProcessReminderEvent", "OrgId", sub.OrgId, "SubscriptionId", sub.Id)
	return s.subscriptionService.SendRenewalReminder(ctx, sub.OrgId, sub.Id)
}
