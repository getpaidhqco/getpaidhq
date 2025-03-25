package factories

import (
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/cart"
	"payloop/internal/lib"
)

type CartFactory struct {
	settingRepository repositories.SettingRepository
	priceRepository   repositories.PriceRepository
	productRepository repositories.ProductRepository
	variantRepository repositories.VariantRepository
	cartRepository    repositories.CartRepository
	logger            logger.Logger
}

func NewCartFactory(
	settingRepository repositories.SettingRepository,
	priceRepository repositories.PriceRepository,
	productRepository repositories.ProductRepository,
	variantRepository repositories.VariantRepository,
	cartRepository repositories.CartRepository,
	logger logger.Logger,
) CartFactory {
	return CartFactory{
		settingRepository: settingRepository,
		priceRepository:   priceRepository,
		productRepository: productRepository,
		variantRepository: variantRepository,
		cartRepository:    cartRepository,
		logger:            logger,
	}
}

func (s CartFactory) NewCart(orgId string, currency common.Currency) cart.Cart {
	instance := cart.NewCart(cart.InitOptions{
		OrgId:             orgId,
		Id:                lib.GenerateId("cart"),
		PriceRepository:   s.priceRepository,
		ProductRepository: s.productRepository,
		VariantRepository: s.variantRepository,
		CartRepository:    s.cartRepository,
		Logger:            s.logger,
	},
		cart.CartData{
			Currency:      string(currency),
			Total:         0,
			SubTotal:      0,
			DiscountTotal: 0,
			ShippingTotal: 0,
			TaxTotal:      0,
			Items:         []cart.Item{},
		})
	return instance
}

func (s CartFactory) NewFromEntity(entity entities.Cart) cart.Cart {
	instance := cart.NewCart(
		cart.InitOptions{
			OrgId:             entity.OrgId,
			Id:                entity.Id,
			PriceRepository:   s.priceRepository,
			ProductRepository: s.productRepository,
			VariantRepository: s.variantRepository,
			CartRepository:    s.cartRepository,
			Logger:            s.logger,
		},
		entity.Data.(cart.CartData),
	)
	return instance
}
