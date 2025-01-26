package services

import (
	"context"
	"errors"
	"github.com/mdwt/payloop-cart/types"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/orders"
	"payloop/internal/lib"
	"time"

	"payloop/internal/repository"
)

type OrderService struct {
	sessionRepository      repository.SessionRepository
	cartRepository         repository.CartRepository
	orderRepository        repository.OrderRepository
	customerRepository     repository.CustomerRepository
	subscriptionRepository repository.SubscriptionRepository
	logger                 lib.Logger
}

func NewOrderService(
	sessionRepository repository.SessionRepository,
	cartRepository repository.CartRepository,
	orderRepository repository.OrderRepository,
	customerRepository repository.CustomerRepository,
	subscriptionRepository repository.SubscriptionRepository,
	logger lib.Logger,
) OrderService {
	return OrderService{
		customerRepository:     customerRepository,
		sessionRepository:      sessionRepository,
		cartRepository:         cartRepository,
		subscriptionRepository: subscriptionRepository,
		orderRepository:        orderRepository,
		logger:                 logger,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input orders.CreateOrderCommand) (entities.Order, error) {
	s.logger.Info("Creating order for cart", "cart", input.CartId)
	accountId := input.AccountId
	orderId := lib.GenerateId("order")

	cart, err := s.cartRepository.FindByID(ctx, accountId, input.CartId)
	if err != nil {
		s.logger.Error("Failed to find cart id ", "id", input.CartId, err.Error())
		return entities.Order{}, errors.New("cart not found")
	}

	customer, err := s.customerRepository.Create(ctx, entities.Customer{
		AccountId: accountId,
		ID:        lib.GenerateId("customer"),
		Name:      input.Customer.Name,
		Email:     input.Customer.Email,
	})
	if err != nil {
		s.logger.Error("Failed to create customer", err.Error())
		return entities.Order{}, err
	}

	order, err := s.orderRepository.Create(ctx, entities.Order{
		AccountId:  accountId,
		Id:         orderId,
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

	// Go through the list of items in the cart and create the subscriptions for each item
	for _, item := range cart.Data.Items {
		if item.Price.Category == types.PriceCategorySubscription {
			// Create a subscription for the item
			sub := entities.NewSubscriptionFromItem(accountId, orderId, item)
			_, err := s.subscriptionRepository.Create(ctx, sub)
			if err != nil {
				s.logger.Error("Failed to create subscription", err.Error())
				return entities.Order{}, err
			}
		}
	}

	return order, nil
}
