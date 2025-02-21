package services

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/prices"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type ProductService struct {
	productRepository repositories.ProductRepository
	variantRepository repositories.VariantRepository
	priceRepository   repositories.PriceRepository
	cartRepository    repositories.CartRepository
	pubsub            events.PubSub
	logger            logger.Logger
}

func NewProductService(
	productRepository repositories.ProductRepository,
	variantRepository repositories.VariantRepository,
	priceRepository repositories.PriceRepository,
	cartRepository repositories.CartRepository,
	logger logger.Logger,
	pubsub events.PubSub,
) ProductService {
	return ProductService{
		productRepository: productRepository,
		variantRepository: variantRepository,
		priceRepository:   priceRepository,
		cartRepository:    cartRepository,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s ProductService) CreateProduct(ctx context.Context, orgId string, request request.CreateProductRequest) (entities.Product, error) {

	product, err := s.productRepository.Create(ctx,
		entities.Product{
			OrgId:       orgId,
			Id:          lib.GenerateId("prod"),
			Name:        request.Name,
			Description: request.Description,
			Metadata:    request.Metadata,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		})
	if err != nil {
		s.logger.Error("Failed to create product", err.Error())
		return entities.Product{}, err
	}

	for _, v := range request.Variants {
		variant, err := s.variantRepository.Create(ctx,
			entities.Variant{
				OrgId:     orgId,
				Id:        lib.GenerateId("var"),
				ProductId: product.Id,
				Name:      v.Name,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			})
		if err != nil {
			s.logger.Error("Failed to create variant", err.Error())
			return entities.Product{}, err
		}

		for _, p := range v.Prices {
			_, err := s.priceRepository.Create(ctx,
				entities.NewPrice(orgId, variant.Id, entities.CreatePriceInput{
					OrgId:              orgId,
					VariantId:          variant.Id,
					Category:           p.Category,
					Scheme:             p.Scheme,
					Cycles:             p.Cycles,
					Currency:           p.Currency,
					UnitPrice:          p.UnitPrice,
					MinPrice:           p.MinPrice,
					SuggestedPrice:     p.SuggestedPrice,
					BillingInterval:    p.BillingInterval,
					BillingIntervalQty: p.BillingIntervalQty,
					TrialInterval:      p.TrialInterval,
					TrialIntervalQty:   p.TrialIntervalQty,
					TaxCode:            p.TaxCode,
					Metadata:           p.Metadata,
				}))
			if err != nil {
				s.logger.Error("Failed to create price", err.Error())
				return entities.Product{}, err
			}
		}
	}

	_ = s.pubsub.Publish(orgId, topic.ProductCreated, product)
	return product, err
}

func (s ProductService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Product, int, error) {
	subs, total, err := s.productRepository.Find(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list products", err.Error())
		return nil, 0, err
	}

	return subs, total, nil
}

func (s ProductService) FindById(ctx context.Context, orgId string, id string) (entities.Product, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to list products", err.Error())
		return entities.Product{}, err
	}

	return product, nil
}

func (s ProductService) CreateProductPrice(ctx context.Context, input entities.CreatePriceInput) (entities.Price, error) {

	if input.BillingInterval == "" {
		input.BillingInterval = prices.BillingIntervalNone
	}
	if input.TrialInterval == "" {
		input.TrialInterval = prices.BillingIntervalNone
	}

	price, err := s.priceRepository.Create(ctx, entities.Price{
		OrgId:              input.OrgId,
		Id:                 lib.GenerateId("price"),
		VariantId:          input.VariantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           common.Currency(input.Currency),
		UnitPrice:          input.UnitPrice,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		Metadata:           input.Metadata,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	})

	if err != nil {
		s.logger.Error("Failed to create product price", err.Error())
		return entities.Price{}, err
	}

	_ = s.pubsub.Publish(input.OrgId, topic.PriceCreated, price)
	return price, nil
}
