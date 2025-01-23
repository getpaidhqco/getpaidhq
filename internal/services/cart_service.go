package services

import (
	"context"
	"payloop/internal/domain/orders"
	"payloop/internal/models"
	"payloop/internal/repository"
)

type CartService struct {
	repo repository.CartRepository
}

func NewCartService(repo repository.CartRepository) CartService {
	return CartService{repo: repo}
}

func (s *CartService) GetOneCart(id uint) (*models.Cart, error) {
	return s.repo.FindByID(context.Background(), id)
}

func (s *CartService) GetAllCarts() ([]*models.Cart, error) {
	return s.repo.FindAll(context.Background())
}

func (s *CartService) CreateCart(ctx context.Context, input orders.CreateCartInput) (models.Cart, error) {
	return s.repo.Create(ctx, input)
}
