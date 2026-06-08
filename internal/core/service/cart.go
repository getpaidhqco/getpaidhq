package service

import (
	"context"
	"fmt"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type CartService struct {
	cartRepository    port.CartRepository
	priceRepository   port.PriceRepository
	productRepository port.ProductRepository
	logger            port.Logger
}

func NewCartService(
	repo port.CartRepository,
	priceRepository port.PriceRepository,
	logger port.Logger,
	productRepository port.ProductRepository,
) *CartService {
	return &CartService{
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
func (s *CartService) AddProduct(ctx context.Context, input port.AddProductCommand) (domain.Cart, error) {

	cartEntity, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return domain.Cart{}, err
	}

	product, err := s.productRepository.FindById(ctx, input.OrgId, input.ProductId)
	if err != nil {
		s.logger.Error("Product doesnt exist", "product_id", input.ProductId, err.Error())
		return domain.Cart{}, lib.NewCustomError(
			lib.NotFoundError, fmt.Sprintf("Product %s not found", input.ProductId),
			err,
		)
	}

	// Archived products are retired and cannot be sold — block adding them to a cart.
	if product.IsArchived() {
		return domain.Cart{}, lib.NewCustomError(
			lib.ConflictError,
			fmt.Sprintf("Product %s is archived and cannot be sold", product.Id),
			nil,
		)
	}

	price, err := s.priceRepository.FindById(ctx, input.OrgId, input.PriceId)
	if err != nil {
		s.logger.Error("Price doesnt exist", "price_id", input.PriceId, err.Error())
		return domain.Cart{}, lib.NewCustomError(
			lib.NotFoundError, fmt.Sprintf("Price %s not found", input.PriceId),
			err,
		)
	}

	cartEntity.Data.Items = append(cartEntity.Data.Items, domain.CartLineItem{
		Id:            lib.GenerateId("ci"),
		ProductId:     product.Id,
		Price:         domain.PriceToCartItemPrice(price),
		Description:   product.Name,
		Quantity:      int64(input.Quantity),
		UnitPrice:     price.UnitPrice,
		SubTotal:      price.UnitPrice * int64(input.Quantity),
		DiscountTotal: 0,
		TaxTotal:      0,
		ShippingTotal: 0,
		Total:         price.UnitPrice * int64(input.Quantity),
	})
	cartEntity.Calculate()

	_, err = s.cartRepository.Update(ctx, cartEntity)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return domain.Cart{}, err
	}

	return cartEntity, nil
}

// RemoveItem removes an item from the cart. It returns updated cart.
func (s *CartService) RemoveItem(ctx context.Context, input port.RemoveItemCommand) (domain.Cart, error) {

	cartEntity, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return domain.Cart{}, err
	}

	cartEntity.RemoveItem(input.Id)

	_, err = s.cartRepository.Update(ctx, cartEntity)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return domain.Cart{}, err
	}

	return cartEntity, nil
}

// AdjustItem adjusts an item in the cart. It returns updated cart.
func (s *CartService) AdjustItem(ctx context.Context, input port.AdjustCommand) (domain.Cart, error) {

	cartEntity, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return domain.Cart{}, err
	}

	err = cartEntity.AdjustQuantity(input.ProductId, int64(input.Quantity))
	if err != nil {
		return domain.Cart{}, err
	}

	_, err = s.cartRepository.Update(ctx, cartEntity)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return domain.Cart{}, err
	}

	return cartEntity, nil
}
