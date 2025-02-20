package services

import (
	"context"
	"errors"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/entities/payments"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/payment_providers"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type OrderService struct {
	sessionRepository      repositories.SessionRepository
	cartRepository         repositories.CartRepository
	orderRepository        repositories.OrderRepository
	customerRepository     repositories.CustomerRepository
	subscriptionRepository repositories.SubscriptionRepository
	orderItemRepository    repositories.OrderItemRepository
	paymentRepository      repositories.PaymentRepository
	gatewayFactory         factories.GatewayFactory
	pubsub                 events.PubSub
	db                     lib.Database
	logger                 logger.Logger
}

func NewOrderService(
	sessionRepository repositories.SessionRepository,
	cartRepository repositories.CartRepository,
	orderRepository repositories.OrderRepository,
	customerRepository repositories.CustomerRepository,
	orderItemRepository repositories.OrderItemRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	paymentRepository repositories.PaymentRepository,
	gatewayFactory factories.GatewayFactory,
	pubsub events.PubSub,
	db lib.Database,
	logger logger.Logger,
) interfaces.OrderService {
	return OrderService{
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		cartRepository:         cartRepository,
		subscriptionRepository: subscriptionRepository,
		orderRepository:        orderRepository,
		logger:                 logger,
		gatewayFactory:         gatewayFactory,
		paymentRepository:      paymentRepository,
		pubsub:                 pubsub,
		db:                     db,

		orderItemRepository: orderItemRepository,
	}
}

func (s OrderService) CreateOrderFromCart(ctx context.Context, input orders.CreateOrderInput) (entities.Order, payment_providers.InitPaymentResponse, error) {
	s.logger.Info("Creating order for cart", "cart", input.CartId)
	orgId := input.OrgId
	orderId := lib.GenerateId("order")

	cart, err := s.cartRepository.FindById(ctx, orgId, input.CartId)
	if err != nil {
		s.logger.Error("Failed to find cart id ", "id", input.CartId, err.Error())
		return entities.Order{}, payment_providers.InitPaymentResponse{}, errors.New("cart not found")
	}

	customer, err := s.customerRepository.Create(ctx, entities.Customer{
		OrgId: orgId,
		Id:    lib.GenerateId("customer"),
		Name:  input.Customer.Name,
		Email: input.Customer.Email,
	})
	if err != nil {
		s.logger.Error("Failed to create customer", err.Error())
		return entities.Order{}, payment_providers.InitPaymentResponse{}, err
	}

	order, err := s.orderRepository.Create(ctx, entities.Order{
		OrgId:      orgId,
		Id:         orderId,
		Reference:  orderId,
		CustomerId: customer.Id,
		Status:     entities.OrderStatusPending,
		SessionId:  "-",
		CartId:     cart.Id,
		Currency:   cart.Data.Currency,
		Total:      cart.Data.Total,
		Metadata:   nil,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	if err != nil {
		s.logger.Error("Failed to create order", err.Error())
		return entities.Order{}, payment_providers.InitPaymentResponse{}, err
	}

	// Go through the list of items in the cart and create the order items for each item
	for _, item := range cart.Data.Items {
		orderItem, err := s.orderItemRepository.Create(ctx, entities.OrderItem{
			OrgId:       orgId,
			Id:          lib.GenerateId("order_item"),
			OrderId:     orderId,
			PriceId:     item.Price.Id,
			Description: item.Description,
			Quantity:    int(item.Quantity),
			Metadata:    nil,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
		if err != nil {
			s.logger.Error("Failed to create order item", "item", item, "err", err.Error())
			return entities.Order{}, payment_providers.InitPaymentResponse{}, err
		}

		if orderItem.Price.Category == prices.PriceCategorySubscription {
			subscription := entities.NewSubscriptionFromOrderItem(orderItem)
			subscription.CustomerId = customer.Id
			subscription.PspId = input.PspId

			_, err := s.subscriptionRepository.Create(ctx, subscription)
			if err != nil {
				s.logger.Error("Failed to create subscription", "item", item, err.Error())
				return entities.Order{}, payment_providers.InitPaymentResponse{}, err
			}
		}
	}

	gw, err := s.gatewayFactory.NewGateway(ctx, orgId, common.Gateway(input.PspId))
	if err != nil {
		s.logger.Error("Failed to get gateway", err.Error())
		return entities.Order{}, payment_providers.InitPaymentResponse{}, err
	}
	// initialise the payment session with the payment processor
	pspResponse, err := gw.InitPayment(ctx, payment_providers.InitPaymentCommand{
		OrgId:    orgId,
		Cart:     cart.Data,
		Order:    order,
		Customer: customer,
	})
	if err != nil {
		s.logger.Error("Failed to initialise payment gateway", err.Error())
		return entities.Order{}, payment_providers.InitPaymentResponse{}, err
	}

	return order, pspResponse, nil
}

// CompleteOrder marks a pending order as completed and updates the subscriptions to reflect any payment received
// This is a special case with orders
// 1. The order is marked as completed
// 2. The subscriptions are updated to reflect the payment received
// 3. A payment is created for the order
// 4. A payment method is created for the customer
// It all happens here for now because it must be part of the same transaction. not sure if this is the best way
// or if we can have transactions in temporal workflows
func (s OrderService) CompleteOrder(ctx context.Context, input orders.CompleteOrderCommand) (entities.Order, error) {
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
	paymentMethod, err := s.customerRepository.CreatePaymentMethod(ctx, entities.PaymentMethod{
		OrgId:      orgId,
		Id:         lib.GenerateId("payment_method"),
		Psp:        input.PaymentContext.Psp,
		Token:      input.PaymentContext.PaymentMethod.Token,
		Name:       "Default",
		CustomerId: order.CustomerId,
		IsDefault:  true,
		BillingAddress: entities.Address{
			Line1: order.Customer.Name,
		},
		Type:      input.PaymentContext.PaymentMethod.Type,
		Details:   input.PaymentContext.PaymentMethod,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
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

			renewsAt := subscription.CalculateNextBillingDate()
			subscription.RenewsAt = renewsAt
			subscription.CurrentPeriodStart = subscription.StartDate
			subscription.CurrentPeriodEnd = renewsAt
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
			PspId:          input.PaymentContext.Payment.PspId,
			OrderId:        orderId,
			SubscriptionId: subscriptionId,
			Status:         payments.PaymentStatusSucceeded,
			Currency:       input.PaymentContext.Payment.Currency,
			Amount:         input.PaymentContext.Payment.Amount,
			PspFee:         0,
			PlatformFee:    0,
			NetAmount:      input.PaymentContext.Payment.Amount,
			Metadata:       nil,
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
