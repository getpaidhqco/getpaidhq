package services

import (
	"context"
	"errors"
	"payloop/internal/application/lib/events"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/domain/entities/prices"
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
	paymentGateway         payment_providers.Gateway
	pubsub                 events.PubSub
	logger                 lib.Logger
}

func NewOrderService(
	sessionRepository repositories.SessionRepository,
	cartRepository repositories.CartRepository,
	orderRepository repositories.OrderRepository,
	customerRepository repositories.CustomerRepository,
	orderItemRepository repositories.OrderItemRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	paymentGateway payment_providers.Gateway,
	pubsub events.PubSub,
	logger lib.Logger,
) OrderService {
	return OrderService{
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		cartRepository:         cartRepository,
		subscriptionRepository: subscriptionRepository,
		orderRepository:        orderRepository,
		logger:                 logger,
		pubsub:                 pubsub,
		paymentGateway:         paymentGateway,
		orderItemRepository:    orderItemRepository,
	}
}

func (s *OrderService) CreateOrderFromCart(ctx context.Context, input orders.CreateOrderInput) (entities.Order, payment_providers.InitPaymentResponse, error) {
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
		orderItem := entities.OrderItem{
			OrgId:       orgId,
			Id:          lib.GenerateId("order_item"),
			OrderId:     orderId,
			PriceId:     item.Price.Id,
			Description: item.Description,
			Quantity:    item.Quantity,
			Metadata:    nil,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		orderItem, err := s.orderItemRepository.Create(ctx, orderItem)

		if orderItem.Price.Category == prices.PriceCategorySubscription {
			subscription := entities.NewSubscriptionFromOrderItem(orderItem)
			_, err := s.subscriptionRepository.Create(ctx, subscription)
			if err != nil {
				s.logger.Error("Failed to create subscription", "item", item, err.Error())
				return entities.Order{}, payment_providers.InitPaymentResponse{}, err
			}
		}

		if err != nil {
			s.logger.Error("Failed to create order item", "item", item, err.Error())
			return entities.Order{}, payment_providers.InitPaymentResponse{}, err
		}
	}

	// initialise the payment session with the payment processor
	pspResponse, err := s.paymentGateway.InitPayment(ctx, payment_providers.InitPaymentCommand{
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
func (s *OrderService) CompleteOrder(ctx context.Context, input orders.CompleteOrderCommand) (entities.Order, error) {
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
		Name:       "Default",
		CustomerId: order.CustomerId,
		IsDefault:  true,
		BillingAddress: entities.Address{
			Line1: order.Customer.Name,
		},
		Type:    input.PaymentContext.PaymentMethod.Type,
		Details: nil,
	})
	if err != nil {
		s.logger.Error("Failed to create payment method", err.Error())
		return entities.Order{}, err
	}
	s.logger.Infof("Created payment method %s for order %s", paymentMethod.Id, order.Id)

	// find subscriptions for the order and update the status to active
	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Order{}, err
	}
	for _, subscription := range subscriptions {
		// TODO this needs to happen but not sure if here or like this
		if input.PaymentContext.Payment.Amount > 0 && subscription.StartDate.Sub(time.Now().UTC()) < 0 {
			subscription.LastCharge = &subscription.StartDate
			subscription.TotalRevenue = subscription.Amount
			subscription.CyclesProcessed = 1
			renewsAt := subscription.NextBillingDate()
			subscription.RenewsAt = &renewsAt
		}

		subscription.PaymentMethodId = &paymentMethod.Id
		subscription.Status = entities.SubscriptionStatusActive
		_, err := s.subscriptionRepository.Update(ctx, subscription)
		if err != nil {
			s.logger.Error("Failed to update subscription status", err.Error())
			return entities.Order{}, err
		}
	}

	// publish order completed event
	_ = s.pubsub.PublishJSON(events.TopicOrderCompleted, order)

	return order, nil
}
