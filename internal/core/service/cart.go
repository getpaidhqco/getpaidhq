package service

import (
	"context"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	// TODO: move cart types out of infrastructure - this import should become a domain/port reference
	cartlib "payloop/internal/infrastructure/cart"
)

type CartService struct {
	cartRepository    port.CartRepository
	priceRepository   port.PriceRepository
	productRepository port.ProductRepository
	cartFactory       *CartFactory
	logger            port.Logger
}

func NewCartService(
	repo port.CartRepository,
	priceRepository port.PriceRepository,
	logger port.Logger,
	cartFactory *CartFactory,
	productRepository port.ProductRepository,
) *CartService {
	return &CartService{
		cartFactory:       cartFactory,
		cartRepository:    repo,
		priceRepository:   priceRepository,
		productRepository: productRepository,
		logger:            logger,
	}
}

func (s *CartService) GetCart(orgId string, id string) (domain.Cart, error) {
	return s.cartRepository.FindById(context.Background(), orgId, id)
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) AddProduct(ctx context.Context, input domain.AddProductCommand) (domain.Cart, error) {

	cartEntity, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return domain.Cart{}, err
	}

	// TODO: resolve cart factory - NewFromEntity depends on infrastructure cart types
	// For now, use the cart library directly with a type assertion
	_ = cartEntity
	_ = cartlib.AddItemInput{
		ProductId: input.ProductId,
		PriceId:   input.PriceId,
		Quantity:  input.Quantity,
	}

	// TODO: complete this once cart types are moved to domain
	// cartInstance := s.cartFactory.NewFromEntity(cartEntity)
	// _, err = cartInstance.AddItem(ctx, cartlib.AddItemInput{...})
	// cartEntity.Data = cartInstance.CartData
	_, err = s.cartRepository.Update(ctx, cartEntity)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return domain.Cart{}, err
	}

	return cartEntity, nil
}

// RemoveItem removes an item from the cart. It returns updated cart.
func (s *CartService) RemoveItem(ctx context.Context, input domain.RemoveItemCommand) (domain.Cart, error) {

	cartEntity, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return domain.Cart{}, err
	}

	// TODO: resolve cart factory - NewFromEntity depends on infrastructure cart types
	// cartInstance := s.cartFactory.NewFromEntity(cartEntity)
	// _, err = cartInstance.RemoveItem(input.Id)
	// cartEntity.Data = cartInstance.CartData
	_, err = s.cartRepository.Update(ctx, cartEntity)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return domain.Cart{}, err
	}

	return cartEntity, nil
}

// AdjustItem adjusts an item in the cart. It returns updated cart.
func (s *CartService) AdjustItem(ctx context.Context, input domain.AdjustCommand) (domain.Cart, error) {

	cartEntity, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return domain.Cart{}, err
	}

	// TODO: resolve cart factory - NewFromEntity depends on infrastructure cart types
	// cartInstance := s.cartFactory.NewFromEntity(cartEntity)
	// _, err = cartInstance.RemoveItem(input.ProductId)
	// cartEntity.Data = cartInstance.CartData
	_, err = s.cartRepository.Update(ctx, cartEntity)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return domain.Cart{}, err
	}

	return cartEntity, nil
}
