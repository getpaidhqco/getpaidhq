package services

import (
	"context"
	"errors"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/entities/payment_methods"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type OrderWorkflowService struct {
	sessionRepository       repositories.SessionRepository
	cartRepository          repositories.CartRepository
	priceRepository         repositories.PriceRepository
	orderRepository         repositories.OrderRepository
	customerRepository      repositories.CustomerRepository
	subscriptionRepository  repositories.SubscriptionRepository
	paymentMethodRepository repositories.PaymentMethodRepository
	orderItemRepository     repositories.OrderItemRepository
	paymentRepository       repositories.PaymentRepository
	gatewayFactory          factories.GatewayFactory
	pubsub                  events.PubSub
	logger                  logger.Logger
}

func NewOrderWorkflowService(
	sessionRepository repositories.SessionRepository,
	priceRepository repositories.PriceRepository,
	cartRepository repositories.CartRepository,
	orderRepository repositories.OrderRepository,
	customerRepository repositories.CustomerRepository,
	paymentMethodRepository repositories.PaymentMethodRepository,
	orderItemRepository repositories.OrderItemRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	paymentRepository repositories.PaymentRepository,
	gatewayFactory factories.GatewayFactory,
	pubsub events.PubSub,
	logger logger.Logger,
) interfaces.OrderWorkflowService {
	return OrderWorkflowService{
		customerRepository:      customerRepository,
		priceRepository:         priceRepository,
		sessionRepository:       sessionRepository,
		paymentMethodRepository: paymentMethodRepository,
		cartRepository:          cartRepository,
		subscriptionRepository:  subscriptionRepository,
		orderRepository:         orderRepository,
		logger:                  logger,
		gatewayFactory:          gatewayFactory,
		paymentRepository:       paymentRepository,
		pubsub:                  pubsub,
		orderItemRepository:     orderItemRepository,
	}
}

// CompleteCheckoutSession marks a pending order as completed and updates the subscriptions to reflect any payment received.
// It is triggered when a payment is received from the payment processor
// This is a special case with orders
// 1. The order is marked as completed
// 2. The subscriptions are updated to reflect the payment received
// 3. A payment is created for the order
// 4. A payment method is created for the customer
// It all happens here for now because it must be part of the same transaction. not sure if this is the best way
// or if we can have transactions in temporal workflows
func (s OrderWorkflowService) CompleteCheckoutSession(ctx context.Context, input orders.CompleteCheckoutSessionInput) (entities.Order, error) {
	s.logger.Info("Completing order", "order_id", input.OrderId)
	orgId := input.OrgId
	orderId := input.OrderId

	order, err := s.orderRepository.FindById(ctx, orgId, orderId)
	if err != nil {
		return entities.Order{}, errors.New("order not found")
	}

	// update the order status
	order.Status = entities.OrderStatusCompleted
	order.UpdatedAt = time.Now()
	_, err = s.orderRepository.Update(ctx, order)
	if err != nil {
		s.logger.Error("Failed to update order", err.Error())
		return entities.Order{}, err
	}

	// create a payment method
	paymentMethod, err := s.paymentMethodRepository.Create(ctx, entities.PaymentMethod{
		OrgId:          orgId,
		Id:             lib.GenerateId("payment_method"),
		Psp:            string(input.PaymentContext.Psp),
		Token:          input.PaymentContext.PaymentMethod.Token,
		Name:           "Default",
		CustomerId:     order.CustomerId,
		BillingAddress: order.Customer.BillingAddress,
		Type:           payment_methods.PaymentMethodType(input.PaymentContext.PaymentMethod.Type),
		Details:        input.PaymentContext.PaymentMethod,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("Failed to create payment method", err.Error())
		return entities.Order{}, err
	}
	s.logger.Infof("Created payment method %s for order %s", paymentMethod.Id, order.Id)

	var subscriptionId string

	// find subscriptions for the order and update the status to active
	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("no subscriptions", err.Error())
	}

	recurringPayment := len(subscriptions) > 0 && input.PaymentContext.Payment.Amount > 0
	for _, subscription := range subscriptions {

		// TODO this needs to happen but not sure if here or like this
		charged := input.PaymentContext.Payment.Amount > 0 && subscription.StartDate.Sub(time.Now().UTC()) < 0
		if charged {
			subscriptionId = subscription.Id
			subscription.SetActivationDates()
			subscription.Status = entities.SubscriptionStatusActive
			subscription.LastCharge = subscription.StartDate
			subscription.TotalRevenue = subscription.Amount
			subscription.CyclesProcessed = 1
		} else {
			subscription.SetActivationDates()
			subscription.Status = entities.SubscriptionStatusTrial
		}
		subscription.PaymentMethodId = paymentMethod.Id

		_, err := s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to update subscription status", err.Error())
			return entities.Order{}, err
		}
	}

	if input.PaymentContext.Payment.Amount > 0 {
		payment := entities.Payment{
			OrgId:          orgId,
			Id:             lib.GenerateId("pmt"),
			Psp:            input.PaymentContext.Psp,
			PspId:          input.PaymentContext.Payment.PspId,
			Reference:      input.PaymentContext.Payment.Reference,
			OrderId:        orderId,
			SubscriptionId: subscriptionId,
			Status:         payments.PaymentStatusSucceeded,
			Recurring:      recurringPayment,
			Currency:       input.PaymentContext.Payment.Currency,
			Amount:         input.PaymentContext.Payment.Amount,
			PspFee:         0,
			PlatformFee:    0,
			NetAmount:      input.PaymentContext.Payment.Amount,
			Metadata:       nil,
			CompletedAt:    input.PaymentContext.Payment.PaidAt,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}
		payment, err := s.paymentRepository.Create(ctx, payment)
		if err != nil {
			s.logger.Error("Failed to create payment", err.Error())
		}
	}

	// publish order completed event
	_ = s.pubsub.Publish(order.OrgId, topic.OrderCompleted, order)

	return order, nil
}
