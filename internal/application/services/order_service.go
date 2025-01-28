package services

import (
	"context"
	"errors"
	"github.com/mdwt/payloop-cart/types"
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
	paymentGateway         payment_providers.Gateway
	logger                 lib.Logger
}

func NewOrderService(
	sessionRepository repositories.SessionRepository,
	cartRepository repositories.CartRepository,
	orderRepository repositories.OrderRepository,
	customerRepository repositories.CustomerRepository,
	subscriptionRepository repositories.SubscriptionRepository,
	paymentGateway payment_providers.Gateway,
	logger lib.Logger,
) OrderService {
	return OrderService{
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		cartRepository:         cartRepository,
		subscriptionRepository: subscriptionRepository,
		orderRepository:        orderRepository,
		logger:                 logger,
		paymentGateway:         paymentGateway,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input orders.CreateOrderCommand) (entities.Order, error) {
	s.logger.Info("Creating order for cart", "cart", input.CartId)
	orgId := input.OrgId
	orderId := lib.GenerateId("order")

	cart, err := s.cartRepository.FindById(ctx, orgId, input.CartId)
	if err != nil {
		s.logger.Error("Failed to find cart id ", "id", input.CartId, err.Error())
		return entities.Order{}, errors.New("cart not found")
	}

	customer, err := s.customerRepository.Create(ctx, entities.Customer{
		OrgId: orgId,
		ID:    lib.GenerateId("customer"),
		Name:  input.Customer.Name,
		Email: input.Customer.Email,
	})
	if err != nil {
		s.logger.Error("Failed to create customer", err.Error())
		return entities.Order{}, err
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
		return entities.Order{}, err
	}

	// Go through the list of items in the cart and create the subscriptions for each item
	for _, item := range cart.Data.Items {
		if item.Price.Category == types.PriceCategorySubscription {
			// Create a subscription for the item
			sub := entities.NewSubscriptionFromItem(orgId, orderId, item)
			_, err := s.subscriptionRepository.Create(ctx, sub)
			if err != nil {
				s.logger.Error("Failed to create subscription", err.Error())
				return entities.Order{}, err
			}
		}
	}

	// initialise the payment session with the payment processor
	err = s.paymentGateway.InitPayment(ctx, payment_providers.InitPaymentCommand{
		OrgId:    orgId,
		Cart:     cart.Data,
		Order:    order,
		Customer: customer,
	})
	if err != nil {
		s.logger.Error("Failed to initialise payment gateway", err.Error())
		return entities.Order{}, err
	}

	return order, nil
}
