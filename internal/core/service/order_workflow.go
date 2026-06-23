package service

import (
	"context"
	"errors"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
	"time"
)

// OrderWorkflowService handles webhook-driven order completion. It does NOT
// hold the workflow engine: this method is invoked from a workflow step, and
// the step is registered with the very engine that dispatches it — so depending
// on the engine here would create a construction-time cycle.
//
// HTTP-driven order completion (which DOES start subscription workflows) lives
// on OrderService.
type OrderWorkflowService struct {
	orderRepository         port.OrderRepository
	customerRepository      port.CustomerRepository
	subscriptionRepository  port.SubscriptionRepository
	paymentMethodRepository port.PaymentMethodRepository
	paymentRepository       port.PaymentRepository
	priceRepository         port.PriceRepository
	pubsub                  port.PubSub
	logger                  port.Logger
}

func NewOrderWorkflowService(
	orderRepository port.OrderRepository,
	customerRepository port.CustomerRepository,
	subscriptionRepository port.SubscriptionRepository,
	paymentMethodRepository port.PaymentMethodRepository,
	paymentRepository port.PaymentRepository,
	priceRepository port.PriceRepository,
	pubsub port.PubSub,
	logger port.Logger,
) *OrderWorkflowService {
	return &OrderWorkflowService{
		orderRepository:         orderRepository,
		customerRepository:      customerRepository,
		subscriptionRepository:  subscriptionRepository,
		paymentMethodRepository: paymentMethodRepository,
		paymentRepository:       paymentRepository,
		priceRepository:         priceRepository,
		pubsub:                  pubsub,
		logger:                  logger,
	}
}

// CompleteCheckoutSession marks a pending order as completed via a payment webhook.
// This handles the PSP-triggered flow (Paystack/Checkout.com webhook -> order completion).
func (s *OrderWorkflowService) CompleteCheckoutSession(ctx context.Context, input port.CompleteCheckoutSessionInput) (domain.Order, error) {
	s.logger.Info("Completing order via checkout session", "order_id", input.OrderId)
	orgId := input.OrgId
	orderId := input.OrderId

	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		return domain.Order{}, errors.New("order not found")
	}

	order.Status = domain.OrderStatusCompleted
	order.UpdatedAt = time.Now()
	_, err = s.orderRepository.Update(ctx, order)
	if err != nil {
		s.logger.Error("Failed to update order", err.Error())
		return domain.Order{}, err
	}

	paymentCtx := input.PaymentContext

	customer, err := s.customerRepository.FindById(ctx, orgId, order.CustomerId)
	if err != nil {
		s.logger.Error("Failed to find customer for order", "customer_id", order.CustomerId, "err", err.Error())
		return domain.Order{}, err
	}

	// Details keeps the PSP's display data but NOT the token — Details is
	// echoed in API responses and events, and the token already lives in the
	// dedicated (redacting) Token field.
	details := paymentCtx.PaymentMethod
	details.Token = ""
	paymentMethod, err := s.paymentMethodRepository.Create(ctx, domain.PaymentMethod{
		OrgId:          orgId,
		Id:             lib.GenerateId("payment_method"),
		Psp:            string(paymentCtx.Psp),
		Token:          domain.Secret(paymentCtx.PaymentMethod.Token),
		Name:           "Default",
		CustomerId:     order.CustomerId,
		BillingAddress: customer.BillingAddress,
		Type:           domain.PaymentMethodType(paymentCtx.PaymentMethod.Type),
		Details:        details,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("Failed to create payment method", err.Error())
		return domain.Order{}, err
	}
	s.logger.Infof("Created payment method %s for order %s", paymentMethod.Id, order.Id)

	var subscriptionId string
	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("error finding subscriptions", err.Error())
	}

	recurringPayment := len(subscriptions) > 0 && paymentCtx.Payment.Amount > 0
	for _, subscription := range subscriptions {
		// The subscription carries its own cadence + trial (derived from its lines),
		// so activation needs no price. Revenue is the recurring fixed base for the
		// first cycle (the subscription stores no amount, ADR 0002).
		charged := paymentCtx.Payment.Amount > 0 && subscription.StartDate.Sub(time.Now().UTC()) < 0
		if charged {
			fixedBase, err := fixedBaseAmount(ctx, s.orderRepository, s.priceRepository, orgId, subscription.Id)
			if err != nil {
				s.logger.Error("Failed to resolve subscription base", "subscription_id", subscription.Id, "err", err.Error())
				return domain.Order{}, err
			}
			subscriptionId = subscription.Id
			subscription.SetActivationDates()
			subscription.Status = domain.SubscriptionStatusActive
			subscription.LastCharge = subscription.StartDate
			subscription.TotalRevenue = fixedBase
			subscription.CyclesProcessed = 1
		} else {
			subscription.SetActivationDates()
			subscription.Status = domain.SubscriptionStatusTrial
		}
		subscription.PaymentMethodId = paymentMethod.Id

		_, err = s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to update subscription status", err.Error())
			return domain.Order{}, err
		}
	}

	if paymentCtx.Payment.Amount > 0 {
		payment := domain.Payment{
			OrgId:          orgId,
			Id:             lib.GenerateId("pmt"),
			Psp:            paymentCtx.Psp,
			PspId:          paymentCtx.Payment.PspId,
			Reference:      paymentCtx.Payment.Reference,
			OrderId:        orderId,
			SubscriptionId: subscriptionId,
			Status:         domain.PaymentStatusSucceeded,
			Recurring:      recurringPayment,
			Currency:       paymentCtx.Payment.Currency,
			Amount:         paymentCtx.Payment.Amount,
			PspFee:         0,
			PlatformFee:    0,
			NetAmount:      paymentCtx.Payment.Amount,
			Metadata:       nil,
			CompletedAt:    paymentCtx.Payment.PaidAt,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		_, err := s.paymentRepository.Create(ctx, payment)
		if err != nil {
			s.logger.Error("Failed to create payment", err.Error())
		}
	}

	_ = s.pubsub.Publish(order.OrgId, port.TopicOrderCompleted, order)
	return order, nil
}
