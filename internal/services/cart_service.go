package services

import (
	"context"
	cartlib "github.com/mdwt/payloop-cart"
	"payloop/internal/domain/carts"
	"payloop/internal/lib"
	"payloop/internal/models"
	"payloop/internal/repository"
)

type CartService struct {
	cartRepository repository.CartRepository
	logger         lib.Logger
}

func NewCartService(repo repository.CartRepository, logger lib.Logger) CartService {

	return CartService{
		cartRepository: repo,
		logger:         logger,
	}
}

// WithTrx enables repository with transaction
func (s *CartService) WithTrx(trxHandle interface{}) *CartService {

	s.cartRepository = *s.cartRepository.WithTrx(trxHandle)
	return s
}
func (s *CartService) GetCart(acctId string, id string) (models.Cart, error) {
	return s.cartRepository.FindByID(context.Background(), acctId, id)
}

func (s *CartService) CreateCart(ctx context.Context, input carts.CreateCartInput) (models.Cart, error) {
	return s.cartRepository.Create(ctx, input)
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) AddProduct(ctx context.Context, input carts.AddProductCommand) (models.Cart, error) {

	cartModel, err := s.cartRepository.FindByID(ctx, input.AccountId, input.CartId)
	if err != nil {
		return models.Cart{}, err
	}
	cart := cartModel.Data

	newCart, err := cart.AddItem(cartlib.Item{
		ID:        lib.GenerateId(`cartitem`),
		ProductId: "prod-1",
		Price: cartlib.Price{
			Id:                 "price-1",
			Category:           string(cartlib.Subscription),
			Scheme:             string(cartlib.Fixed),
			Currency:           "USD",
			UnitPrice:          1000,
			BillingInterval:    string(cartlib.BillingIntervalMonth),
			BillingIntervalQty: 1,
			TrialInterval:      string(cartlib.BillingIntervalNone),
			TrialIntervalQty:   0,
			TaxCode:            "exempt",
		},
		Description: "New Product",
		Quantity:    input.Quantity,
	})
	if err != nil {
		s.logger.Error(`failed to add product to cart`, err)
		return models.Cart{}, err
	}

	cartModel.Data = *newCart
	_, err = s.cartRepository.Update(ctx, cartModel)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return models.Cart{}, err
	}

	return cartModel, nil
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) RemoveItem(ctx context.Context, input carts.RemoveItemCommand) (models.Cart, error) {

	cartModel, err := s.cartRepository.FindByID(ctx, input.AccountId, input.CartId)
	if err != nil {
		return models.Cart{}, err
	}
	cart := cartModel.Data

	newCart, err := cart.RemoveItem(input.Id)
	if err != nil {
		s.logger.Error(`failed to remove product`, err)
		return models.Cart{}, err
	}

	cartModel.Data = *newCart
	_, err = s.cartRepository.Update(ctx, cartModel)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return models.Cart{}, err
	}

	return cartModel, nil
}

// AddProduct adds product to cart. It returns updated cart.
func (s *CartService) AdjustItem(ctx context.Context, input carts.AdjustCommand) (models.Cart, error) {

	cartModel, err := s.cartRepository.FindByID(ctx, input.AccountId, input.CartId)
	if err != nil {
		return models.Cart{}, err
	}
	cart := cartModel.Data

	newCart, err := cart.AddItem(cartlib.Item{
		ID:        lib.GenerateId(`cartitem`),
		ProductId: "prod-1",
		Price: cartlib.Price{
			Id:                 "price-1",
			Category:           string(cartlib.Subscription),
			Scheme:             string(cartlib.Fixed),
			Currency:           "USD",
			UnitPrice:          1000,
			BillingInterval:    string(cartlib.BillingIntervalMonth),
			BillingIntervalQty: 1,
			TrialInterval:      string(cartlib.BillingIntervalNone),
			TrialIntervalQty:   0,
			TaxCode:            "exempt",
		},
		Description: "New Product",
		Quantity:    input.Quantity,
	})
	if err != nil {
		s.logger.Error(`failed to add product to cart`, err)
		return models.Cart{}, err
	}

	cartModel.Data = *newCart
	_, err = s.cartRepository.Update(ctx, cartModel)
	if err != nil {
		s.logger.Error(`failed to update cart`, err)
		return models.Cart{}, err
	}

	return cartModel, nil
}
