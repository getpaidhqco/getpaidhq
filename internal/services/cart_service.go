package services

import (
	"context"
	"errors"
	cartlib "github.com/mdwt/payloop-cart"
	carttypes "github.com/mdwt/payloop-cart/types"
	"payloop/internal/domain/carts"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
	"payloop/internal/repository"
)

type CartService struct {
	cartRepository    repository.CartRepository
	priceRepository   repository.PriceRepository
	productRepository repository.ProductRepository
	logger            lib.Logger
}

func NewCartService(repo repository.CartRepository,
	priceRepository repository.PriceRepository,
	logger lib.Logger,
	productRepository repository.ProductRepository,
) CartService {
	return CartService{
		cartRepository:    repo,
		priceRepository:   priceRepository,
		productRepository: productRepository,
		logger:            logger,
	}
}

// WithTrx enables repository with transaction
func (s *CartService) WithTrx(trxHandle interface{}) *CartService {

	s.cartRepository = *s.cartRepository.WithTrx(trxHandle)
	return s
}
func (s *CartService) GetCart(acctId string, id string) (entities.Cart, error) {
	return s.cartRepository.FindByID(context.Background(), acctId, id)
}

func (s *CartService) CreateCart(ctx context.Context, input carts.CreateCartInput) (entities.Cart, error) {
	return s.cartRepository.Create(ctx, input)
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) AddProduct(ctx context.Context, input carts.AddProductCommand) (entities.Cart, error) {

	cartModel, err := s.cartRepository.FindByID(ctx, input.AccountId, input.CartId)
	if err != nil {
		return entities.Cart{}, err
	}
	cart := cartModel.Data
	s.logger.Debug(`Found cart`, `id`, cart.Id)

	price, err := s.priceRepository.FindByID(ctx, input.AccountId, input.PriceId)
	if err != nil {
		s.logger.Error(`failed to find price`, err)
		return entities.Cart{}, err
	}
	product, err := s.productRepository.FindByID(ctx, input.AccountId, input.ProductId)
	if err != nil {
		s.logger.Error(`invalid product`, err.Error())
		return entities.Cart{}, errors.New(`invalid product`)
	}

	newCart, err := cart.AddItem(cartlib.Item{
		ID:          lib.GenerateId(`cartitem`),
		ProductId:   product.Id,
		Price:       price.ToCartItemPrice(),
		Description: product.Name,
		Quantity:    input.Quantity,
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

	cartModel, err := s.cartRepository.FindByID(ctx, input.AccountId, input.CartId)
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

	cartModel, err := s.cartRepository.FindByID(ctx, input.AccountId, input.CartId)
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
		Quantity:    input.Quantity,
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
