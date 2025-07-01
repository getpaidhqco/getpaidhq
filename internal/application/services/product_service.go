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
			// Convert request tiers to entity tiers
			var tiers []entities.CreatePriceTierInput
			for _, t := range p.Tiers {
				tiers = append(tiers, entities.CreatePriceTierInput{
					Tier:        t.Tier,
					FromQty:     t.FromQty,
					ToQty:       t.ToQty,
					UnitPrice:   t.UnitPrice,
					Description: t.Description,
				})
			}

			priceInput := entities.CreatePriceInput{
					OrgId:              orgId,
					Label:              p.Label,
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
					HasUsage:           p.HasUsage,
					UsageType:          p.UsageType,
					UnitType:           p.UnitType,
					AggregationType:    p.AggregationType,
					PercentageRate:     p.PercentageRate,
					FixedFee:           p.FixedFee,
					OverageUnitPrice:   p.OverageUnitPrice,
					IncludedUsage:      p.IncludedUsage,
					UsageLimit:         p.UsageLimit,
					Tiers:              tiers,
					Metadata:           p.Metadata,
				}

				price, err := entities.NewPrice(orgId, variant.Id, priceInput)
				if err != nil {
					s.logger.Error("Failed to validate price", err.Error())
					return entities.Product{}, lib.NewCustomError(lib.BadRequestError, err.Error(), nil)
				}

				_, err = s.priceRepository.Create(ctx, price)
			if err != nil {
				s.logger.Error("Failed to create price", err.Error())
				return entities.Product{}, err
			}
		}
	}

	product, err = s.FindById(ctx, orgId, product.Id)
	if err != nil {
		s.logger.Error("Failed to find product", err.Error())
		return entities.Product{}, err
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

	// Create and validate price entity
	price, err := entities.NewPrice(input.OrgId, input.VariantId, input)
	if err != nil {
		s.logger.Error("Failed to validate price", err.Error())
		return entities.Price{}, lib.NewCustomError(lib.BadRequestError, err.Error(), nil)
	}

	// Create price in repository
	price, err = s.priceRepository.Create(ctx, price)
	if err != nil {
		s.logger.Error("Failed to create product price", err.Error())
		return entities.Price{}, err
	}

	_ = s.pubsub.Publish(input.OrgId, topic.PriceCreated, price)
	return price, nil
}

func (s ProductService) UpdateProduct(ctx context.Context, orgId string, id string, request request.UpdateProductRequest) (entities.Product, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find product", err.Error())
		return entities.Product{}, err
	}

	product.Name = request.Name
	product.Description = request.Description
	product.Metadata = request.Metadata
	product.UpdatedAt = time.Now().UTC()

	product, err = s.productRepository.Update(ctx, product)
	if err != nil {
		s.logger.Error("Failed to update product", err.Error())
		return entities.Product{}, err
	}

	_ = s.pubsub.Publish(orgId, topic.ProductUpdated, product)
	return product, nil
}

func (s ProductService) DeleteProduct(ctx context.Context, orgId string, id string) error {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find product", err.Error())
		return err
	}

	err = s.productRepository.Delete(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to delete product", err.Error())
		return err
	}

	_ = s.pubsub.Publish(orgId, topic.ProductDeleted, product)
	return nil
}

func (s ProductService) CreateVariant(ctx context.Context, orgId string, productId string, request request.CreateVariantRequest) (entities.Variant, error) {
	variant, err := s.variantRepository.Create(ctx, entities.Variant{
		OrgId:       orgId,
		Id:          lib.GenerateId("var"),
		ProductId:   productId,
		Name:        request.Name,
		Description: request.Description,
		Metadata:    request.Metadata,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("Failed to create variant", err.Error())
		return entities.Variant{}, err
	}

	_ = s.pubsub.Publish(orgId, topic.VariantCreated, variant)
	return variant, nil
}

func (s ProductService) GetVariant(ctx context.Context, orgId string, id string) (entities.Variant, error) {
	variant, err := s.variantRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find variant", err.Error())
		return entities.Variant{}, err
	}

	return variant, nil
}

func (s ProductService) ListVariants(ctx context.Context, orgId string, productId string, pagination request.Pagination) ([]entities.Variant, int, error) {
	variants, total, err := s.variantRepository.FindByProductId(ctx, orgId, productId, pagination)
	if err != nil {
		s.logger.Error("Failed to list variants", err.Error())
		return nil, 0, err
	}

	return variants, total, nil
}

func (s ProductService) UpdateVariant(ctx context.Context, orgId string, id string, request request.UpdateVariantRequest) (entities.Variant, error) {
	variant, err := s.variantRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find variant", err.Error())
		return entities.Variant{}, err
	}

	variant.Name = request.Name
	variant.Description = request.Description
	variant.Metadata = request.Metadata
	variant.UpdatedAt = time.Now().UTC()

	variant, err = s.variantRepository.Update(ctx, variant)
	if err != nil {
		s.logger.Error("Failed to update variant", err.Error())
		return entities.Variant{}, err
	}

	_ = s.pubsub.Publish(orgId, topic.VariantUpdated, variant)
	return variant, nil
}

func (s ProductService) DeleteVariant(ctx context.Context, orgId string, id string) error {
	variant, err := s.variantRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find variant", err.Error())
		return err
	}

	err = s.variantRepository.Delete(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to delete variant", err.Error())
		return err
	}

	_ = s.pubsub.Publish(orgId, topic.VariantDeleted, variant)
	return nil
}

func (s ProductService) GetPrice(ctx context.Context, orgId string, id string) (entities.Price, error) {
	price, err := s.priceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find price", err.Error())
		return entities.Price{}, err
	}

	return price, nil
}

func (s ProductService) ListPrices(ctx context.Context, orgId string, variantId string, pagination request.Pagination) ([]entities.Price, int, error) {
	prices, total, err := s.priceRepository.FindByVariantId(ctx, orgId, variantId, pagination)
	if err != nil {
		s.logger.Error("Failed to list prices", err.Error())
		return nil, 0, err
	}

	return prices, total, nil
}

func (s ProductService) UpdatePrice(ctx context.Context, orgId string, id string, input entities.CreatePriceInput) (entities.Price, error) {
	price, err := s.priceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find price", err.Error())
		return entities.Price{}, err
	}

	if input.BillingInterval == "" {
		input.BillingInterval = prices.BillingIntervalNone
	}
	if input.TrialInterval == "" {
		input.TrialInterval = prices.BillingIntervalNone
	}

	price.Label = input.Label
	price.Category = input.Category
	price.Scheme = input.Scheme
	price.Cycles = input.Cycles
	price.Currency = common.Currency(input.Currency)
	price.UnitPrice = input.UnitPrice
	price.MinPrice = input.MinPrice
	price.SuggestedPrice = input.SuggestedPrice
	price.BillingInterval = input.BillingInterval
	price.BillingIntervalQty = input.BillingIntervalQty
	price.TrialInterval = input.TrialInterval
	price.TrialIntervalQty = input.TrialIntervalQty
	price.TaxCode = input.TaxCode
	price.Metadata = input.Metadata
	price.UpdatedAt = time.Now().UTC()

	// Convert input tiers to price tiers
	var tiers []entities.PriceTier
	for _, tierInput := range input.Tiers {
		tiers = append(tiers, entities.NewPriceTier(entities.CreatePriceTierInput{
			OrgId:       orgId,
			PriceId:     price.Id,
			Tier:        tierInput.Tier,
			FromQty:     tierInput.FromQty,
			ToQty:       tierInput.ToQty,
			UnitPrice:   tierInput.UnitPrice,
			Description: tierInput.Description,
		}))
	}
	price.Tiers = tiers

	// Validate price before updating
	if err := price.Validate(); err != nil {
		return entities.Price{}, lib.NewCustomError(lib.BadRequestError, err.Error(), nil)
	}

	price, err = s.priceRepository.Update(ctx, price)
	if err != nil {
		s.logger.Error("Failed to update price", err.Error())
		return entities.Price{}, err
	}

	// Update price tiers if any
	if len(input.Tiers) > 0 {
		err = s.priceRepository.UpdatePriceTiers(ctx, orgId, price.Id, price.Tiers)
		if err != nil {
			s.logger.Error("Failed to update price tiers", err.Error())
			return entities.Price{}, err
		}
	} else {
		// Delete all tiers if none are provided
		err = s.priceRepository.DeletePriceTiers(ctx, orgId, price.Id)
		if err != nil {
			s.logger.Error("Failed to delete price tiers", err.Error())
			return entities.Price{}, err
		}
	}

	_ = s.pubsub.Publish(orgId, topic.PriceUpdated, price)
	return price, nil
}

func (s ProductService) DeletePrice(ctx context.Context, orgId string, id string) error {
	price, err := s.priceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find price", err.Error())
		return err
	}

	err = s.priceRepository.Delete(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to delete price", err.Error())
		return err
	}

	_ = s.pubsub.Publish(orgId, topic.PriceDeleted, price)
	return nil
}
