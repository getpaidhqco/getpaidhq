package services

import (
	"context"
	"payloop/internal/models"
	"payloop/internal/repositories"
)

type OrderService struct {
	repo repositories.OrderRepository
}

func NewOrderService(repo repositories.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) GetOneOrder(id uint) (*models.Order, error) {
	return s.repo.FindByID(context.Background(), id)
}

func (s *OrderService) GetAllOrders() ([]*models.Order, error) {
	return s.repo.FindAll(context.Background())
}

func (s *OrderService) CreateOrder(order models.Order) error {
	return s.repo.Create(context.Background(), order)
}

func (s *OrderService) UpdateOrder(order models.Order) error {
	return s.repo.Update(context.Background(), order)
}

func (s *OrderService) DeleteOrder(id uint) error {
	return s.repo.Delete(context.Background(), id)
}
