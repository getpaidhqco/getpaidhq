package services

import (
	"context"
	"payloop/internal/domain/orders"
	"payloop/internal/models"
	"payloop/internal/repository"
)

type OrderService struct {
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) OrderService {
	return OrderService{repo: repo}
}

func (s *OrderService) GetOneOrder(id uint) (*models.Order, error) {
	return s.repo.FindByID(context.Background(), id)
}

func (s *OrderService) GetAllOrders() ([]*models.Order, error) {
	return s.repo.FindAll(context.Background())
}

func (s *OrderService) CreateOrder(ctx context.Context, input orders.CreateOrderInput) (models.Order, error) {
	return s.repo.Create(ctx, input)
}
