package services

import (
	"context"
	"errors"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/orders"
	"payloop/internal/lib"
	"time"

	"payloop/internal/repository"
)

type OrderService struct {
	sessionRepository  repository.SessionRepository
	cartRepository     repository.CartRepository
	orderRepository    repository.OrderRepository
	customerRepository repository.CustomerRepository
	logger             lib.Logger
}

func NewOrderService(
	sessionRepository repository.SessionRepository,
	cartRepository repository.CartRepository,
	orderRepository repository.OrderRepository,
	customerRepository repository.CustomerRepository,
	logger lib.Logger,
) OrderService {
	return OrderService{
		customerRepository: customerRepository,
		sessionRepository:  sessionRepository,
		cartRepository:     cartRepository,
		orderRepository:    orderRepository,
		logger:             logger,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, input orders.CreateOrderCommand) (entities.Order, error) {
	s.logger.Info("Creating order for cart", "cart", input.CartId)
	accountId := input.AccountId

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

	orderId := lib.GenerateId("order")

	return s.orderRepository.Create(ctx, entities.Order{
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
}
