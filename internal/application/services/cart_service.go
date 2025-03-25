package services

import (
	"context"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/carts"
	"payloop/internal/domain/factories"
	"payloop/internal/domain/repositories"
	cartlib "payloop/internal/infrastructure/cart"
)

type CartService struct {
	cartRepository    repositories.CartRepository
	priceRepository   repositories.PriceRepository
	productRepository repositories.ProductRepository
	cartFactory       factories.CartFactory
	logger            logger.Logger
}

func NewCartService(repo repositories.CartRepository,
	priceRepository repositories.PriceRepository,
	logger logger.Logger,
	cartFactory factories.CartFactory,
	productRepository repositories.ProductRepository,
) CartService {
	return CartService{
		cartFactory:       cartFactory,
		cartRepository:    repo,
		priceRepository:   priceRepository,
		productRepository: productRepository,
		logger:            logger,
	}
}

func (s *CartService) GetCart(org_id string, id string) (entities.Cart, error) {
	return s.cartRepository.FindById(context.Background(), org_id, id)
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) AddProduct(ctx context.Context, input carts.AddProductCommand) (entities.Cart, error) {

	cartEntity, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return entities.Cart{}, err
	}
	cartInstance := s.cartFactory.NewFromEntity(cartEntity)

	_, err = cartInstance.AddItem(ctx, cartlib.AddItemInput{
		ProductId: input.ProductId,
		PriceId:   input.PriceId,
		Quantity:  input.Quantity,
	})
	if err != nil {
		s.logger.Error(`failed to add product to cart`, err)
		return entities.Cart{}, err
	}

	cartEntity.Data = cartInstance.CartData
	_, err = s.cartRepository.Update(ctx, cartEntity)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return entities.Cart{}, err
	}

	return cartEntity, nil
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) RemoveItem(ctx context.Context, input carts.RemoveItemCommand) (entities.Cart, error) {

	cartEntity, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return entities.Cart{}, err
	}
	cartInstance := s.cartFactory.NewFromEntity(cartEntity)

	_, err = cartInstance.RemoveItem(input.Id)
	if err != nil {
		s.logger.Error(`failed to add product to cart`, err)
		return entities.Cart{}, err
	}

	cartEntity.Data = cartInstance.CartData
	_, err = s.cartRepository.Update(ctx, cartEntity)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return entities.Cart{}, err
	}

	return cartEntity, nil
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) AdjustItem(ctx context.Context, input carts.AdjustCommand) (entities.Cart, error) {

	cartEntity, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return entities.Cart{}, err
	}
	cartInstance := s.cartFactory.NewFromEntity(cartEntity)

	_, err = cartInstance.RemoveItem(input.ProductId)
	if err != nil {
		s.logger.Error(`failed to add product to cart`, err)
		return entities.Cart{}, err
	}

	cartEntity.Data = cartInstance.CartData
	_, err = s.cartRepository.Update(ctx, cartEntity)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return entities.Cart{}, err
	}

	return cartEntity, nil
}
