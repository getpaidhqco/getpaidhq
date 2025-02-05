package services

import (
	"context"
	"errors"
	"payloop/internal/application/lib/events"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/orders"
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

func (s *OrderService) CreateOrder(ctx context.Context, input orders.CreateOrderCommand) (entities.Order, payment_providers.InitPaymentResponse, error) {
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
		ID:    lib.GenerateId("customer"),
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
		CustomerId: customer.ID,
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
		_, err := s.orderItemRepository.Create(ctx, orderItem)
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

// CompleteOrder marks a pending order as completed and activates any subscriptions
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

	subscriptions, err := s.subscriptionRepository.FindByOrderId(ctx, orgId, orderId)
	if err != nil {
		s.logger.Error("Failed to find subscriptions", err.Error())
		return entities.Order{}, err
	}

	for _, subscription := range subscriptions {
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
