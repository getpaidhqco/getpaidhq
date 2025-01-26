package services

import (
	"context"
	"payloop/internal/domain/orders"
	"payloop/internal/lib"
	"payloop/internal/models"
	"payloop/internal/repository"
)

type OrderService struct {
	sessionRepository repository.SessionRepository
	orderRepository   repository.OrderRepository
	logger            lib.Logger
}

func NewOrderService(repo repository.OrderRepository) OrderService {
	return OrderService{orderRepository: repo}
}

func (s *OrderService) GetOneOrder(id uint) (*models.Order, error) {
	return s.orderRepository.FindByID(context.Background(), id)
}

func (s *OrderService) GetAllOrders() ([]*models.Order, error) {
	return s.orderRepository.FindAll(context.Background())
}

func (s *OrderService) CreateOrder(ctx context.Context, input orders.CreateOrderInput) (models.Order, error) {
	s.logger.Info("Creating order")
	accountId := input.AccountId

	session, err := s.sessionRepository.FindById(ctx, accountId, input.SessionId)
	if err != nil {
		s.logger.Error("Failed to find sessionRepository", err)
		return models.Order{}, err
	}

	createOrderInput := orders.CreateOrderRow{
		AccountId: accountId,
		Customer:  orders.CustomerInput{},
		SessionId: session.Id,
		Currency:  "USD",
		Metadata:  nil,
	}

	return s.orderRepository.Create(ctx, createOrderInput)
}
