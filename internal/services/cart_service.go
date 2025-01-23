package services

import (
	"context"
	"payloop/internal/domain/cart"
	"payloop/internal/models"
	"payloop/internal/repository"
)

type CartService struct {
	repo repository.CartRepository
}

func NewCartService(repo repository.CartRepository) CartService {
	return CartService{repo: repo}
}

func (s *CartService) GetCart(acctId string, id string) (models.Cart, error) {
	return s.repo.FindByID(context.Background(), acctId, id)
}

func (s *CartService) CreateCart(ctx context.Context, input cart.CreateCartInput) (models.Cart, error) {
	return s.repo.Create(ctx, input)
}
