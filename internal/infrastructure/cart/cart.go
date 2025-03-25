package cart

import (
	"context"
	"fmt"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type InitOptions struct {
	OrgId             string
	Id                string
	PriceRepository   repositories.PriceRepository
	ProductRepository repositories.ProductRepository
	VariantRepository repositories.VariantRepository
	CartRepository    repositories.CartRepository
	Logger            logger.Logger
}

type Cart struct {
	OrgId string
	Id    string
	CartData
	priceRepository   repositories.PriceRepository
	productRepository repositories.ProductRepository
	variantRepository repositories.VariantRepository
	cartRepository    repositories.CartRepository
	logger            logger.Logger
}

func NewCart(init InitOptions, cartData CartData) Cart {

	return Cart{
		OrgId:             init.OrgId,
		Id:                init.Id,
		CartData:          cartData,
		priceRepository:   init.PriceRepository,
		productRepository: init.ProductRepository,
		variantRepository: init.VariantRepository,
		cartRepository:    init.CartRepository,
		logger:            init.Logger,
	}
}

func (c *Cart) AddItem(ctx context.Context, input AddItemInput) (*Cart, error) {
	product, err := c.productRepository.FindById(ctx, c.OrgId, input.ProductId)
	if err != nil {
		c.logger.Error("Product doesnt exist", "product_id", input.ProductId, err.Error())
		return c, lib.NewCustomError(
			lib.NotFoundError, fmt.Sprintf("Product %s not found", input.ProductId),
			err,
		)
	}
	price, err := c.priceRepository.FindById(ctx, c.OrgId, input.PriceId)
	if err != nil {
		c.logger.Error("Price doesnt exist", "price_id", input.PriceId, err.Error())
		return c, lib.NewCustomError(
			lib.NotFoundError, fmt.Sprintf("Price %s not found", input.PriceId),
			err,
		)
	}

	c.Items = append(c.Items, Item{
		Id:            lib.GenerateId("ci"),
		ProductId:     product.Id,
		Price:         PriceToCartItemPrice(price),
		Description:   product.Name,
		Quantity:      int64(input.Quantity),
		UnitPrice:     price.UnitPrice,
		SubTotal:      price.UnitPrice * int64(input.Quantity),
		DiscountTotal: 0,
		TaxTotal:      0,
		ShippingTotal: 0,
		Total:         price.UnitPrice * int64(input.Quantity),
	})
	return c.Calculate()
}

func (c *Cart) AdjustQuantity(id string, quantity int64) (*Cart, error) {
	for i, item := range c.Items {
		if item.Id == id {
			c.Items[i].Quantity = quantity
			return c.Calculate()
		}
	}
	return nil, fmt.Errorf("item not found")
}

func (c *Cart) RemoveItem(id string) (*Cart, error) {
	for i, item := range c.Items {
		if item.Id == id {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			break
		}
	}
	return c.Calculate()
}

func (c *Cart) Calculate() (*Cart, error) {
	var total int64 = 0
	var subTotal int64 = 0
	var discountTotal int64 = 0
	var taxTotal int64 = 0

	for _, item := range c.Items {
		item.Calculate()
		discountTotal += item.DiscountTotal
		taxTotal += item.TaxTotal
		subTotal += item.SubTotal
		total = subTotal
	}

	c.Total = total
	c.SubTotal = subTotal
	c.TaxTotal = taxTotal
	c.DiscountTotal = discountTotal

	return c, nil
}
