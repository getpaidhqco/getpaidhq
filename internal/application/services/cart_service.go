package services

import (
	"context"
	"errors"
	cartlib "github.com/mdwt/payloop-cart"
	carttypes "github.com/mdwt/payloop-cart/types"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/carts"
	"payloop/internal/domain/repositories"

	"payloop/internal/lib"
)

type CartService struct {
	cartRepository    repositories.CartRepository
	priceRepository   repositories.PriceRepository
	productRepository repositories.ProductRepository
	logger            logger.Logger
}

func NewCartService(repo repositories.CartRepository,
	priceRepository repositories.PriceRepository,
	logger logger.Logger,
	productRepository repositories.ProductRepository,
) CartService {
	return CartService{
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

	cartModel, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return entities.Cart{}, err
	}
	cart := cartModel.Data
	s.logger.Debug(`Found cart`, `id`, cart.Id)

	price, err := s.priceRepository.FindById(ctx, input.OrgId, input.PriceId)
	if err != nil {
		return entities.Cart{}, lib.NewCustomError(lib.NotFoundError, "Price not found", err)
	}
	product, err := s.productRepository.FindById(ctx, input.OrgId, input.ProductId)
	if err != nil {
		s.logger.Error(`invalid product`, err.Error())
		return entities.Cart{}, errors.New(`invalid product`)
	}

	newCart, err := cart.AddItem(cartlib.Item{
		ID:          lib.GenerateId(`cartitem`),
		ProductId:   product.Id,
		Price:       price.ToCartItemPrice(),
		Description: product.Name,
		Quantity:    int64(input.Quantity),
	})
	if err != nil {
		s.logger.Error(`failed to add product to cart`, err)
		return entities.Cart{}, err
	}

	cartModel.Data = *newCart
	_, err = s.cartRepository.Update(ctx, cartModel)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return entities.Cart{}, err
	}

	return cartModel, nil
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) RemoveItem(ctx context.Context, input carts.RemoveItemCommand) (entities.Cart, error) {

	cartModel, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return entities.Cart{}, err
	}
	cart := cartModel.Data

	newCart, err := cart.RemoveItem(input.Id)
	if err != nil {
		s.logger.Error(`failed to remove product`, err)
		return entities.Cart{}, err
	}

	cartModel.Data = *newCart
	_, err = s.cartRepository.Update(ctx, cartModel)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return entities.Cart{}, err
	}

	return cartModel, nil
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) AdjustItem(ctx context.Context, input carts.AdjustCommand) (entities.Cart, error) {

	cartModel, err := s.cartRepository.FindById(ctx, input.OrgId, input.CartId)
	if err != nil {
		return entities.Cart{}, err
	}
	cart := cartModel.Data

	newCart, err := cart.AddItem(cartlib.Item{
		ID:        lib.GenerateId(`cartitem`),
		ProductId: "prod-1",
		Price: cartlib.Price{
			Id:                 "price-1",
			Category:           carttypes.PriceCategorySubscription,
			Scheme:             carttypes.Fixed,
			Currency:           "USD",
			UnitPrice:          1000,
			BillingInterval:    carttypes.BillingIntervalMonth,
			BillingIntervalQty: 1,
			TrialInterval:      carttypes.BillingIntervalNone,
			TrialIntervalQty:   0,
			TaxCode:            "exempt",
		},
		Description: "New Product",
		Quantity:    int64(input.Quantity),
	})
	if err != nil {
		s.logger.Error(`failed to add product to cart`, err)
		return entities.Cart{}, err
	}

	cartModel.Data = *newCart
	_, err = s.cartRepository.Update(ctx, cartModel)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return entities.Cart{}, err
	}

	return cartModel, nil
}
