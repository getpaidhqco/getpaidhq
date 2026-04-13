package service

import (
	"context"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
	"time"
)

type ProductService struct {
	productRepository port.ProductRepository
	variantRepository port.VariantRepository
	priceRepository   port.PriceRepository
	cartRepository    port.CartRepository
	pubsub            port.PubSub
	logger            port.Logger
}

func NewProductService(
	productRepository port.ProductRepository,
	variantRepository port.VariantRepository,
	priceRepository port.PriceRepository,
	cartRepository port.CartRepository,
	logger port.Logger,
	pubsub port.PubSub,
) *ProductService {
	return &ProductService{
		productRepository: productRepository,
		variantRepository: variantRepository,
		priceRepository:   priceRepository,
		cartRepository:    cartRepository,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, orgId string, input domain.CreateProductInput) (domain.Product, error) {

	product, err := s.productRepository.Create(ctx,
		domain.Product{
			OrgId:       orgId,
			Id:          lib.GenerateId("prod"),
			Name:        input.Name,
			Description: input.Description,
			Metadata:    input.Metadata,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		})
	if err != nil {
		s.logger.Error("failed to create product", "error", err)
		return domain.Product{}, err
	}

	for _, v := range input.Variants {
		variant, err := s.variantRepository.Create(ctx,
			domain.Variant{
				OrgId:     orgId,
				Id:        lib.GenerateId("var"),
				ProductId: product.Id,
				Name:      v.Name,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			})
		if err != nil {
			s.logger.Error("failed to create variant", "error", err)
			return domain.Product{}, err
		}

		for _, p := range v.Prices {
			_, err := s.priceRepository.Create(ctx,
				domain.NewPrice(orgId, variant.Id, domain.CreatePriceInput{
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
					Metadata:           p.Metadata,
				}))
			if err != nil {
				s.logger.Error("failed to create price", "error", err)
				return domain.Product{}, err
			}
		}
	}

	product, err = s.FindById(ctx, orgId, product.Id)
	if err != nil {
		s.logger.Error("failed to find product", "error", err)
		return domain.Product{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicProductCreated, product)
	return product, err
}

func (s *ProductService) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.Product, int, error) {
	subs, total, err := s.productRepository.Find(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("failed to list products", "error", err)
		return nil, 0, err
	}

	return subs, total, nil
}

func (s *ProductService) FindById(ctx context.Context, orgId string, id string) (domain.Product, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find product", "error", err)
		return domain.Product{}, err
	}

	return product, nil
}

func (s *ProductService) CreateProductPrice(ctx context.Context, input domain.CreatePriceInput) (domain.Price, error) {

	if input.BillingInterval == "" {
		input.BillingInterval = domain.BillingIntervalNone
	}
	if input.TrialInterval == "" {
		input.TrialInterval = domain.BillingIntervalNone
	}

	price, err := s.priceRepository.Create(ctx, domain.Price{
		OrgId:              input.OrgId,
		Id:                 lib.GenerateId("price"),
		Label:              input.Label,
		VariantId:          input.VariantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           domain.Currency(input.Currency),
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
		s.logger.Error("failed to create product price", "error", err)
		return domain.Price{}, err
	}

	_ = s.pubsub.Publish(input.OrgId, port.TopicPriceCreated, price)
	return price, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, orgId string, id string, input domain.UpdateProductInput) (domain.Product, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find product", "error", err)
		return domain.Product{}, err
	}

	product.Name = input.Name
	product.Description = input.Description
	product.Metadata = input.Metadata
	product.UpdatedAt = time.Now().UTC()

	product, err = s.productRepository.Update(ctx, product)
	if err != nil {
		s.logger.Error("failed to update product", "error", err)
		return domain.Product{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicProductUpdated, product)
	return product, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, orgId string, id string) error {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find product", "error", err)
		return err
	}

	err = s.productRepository.Delete(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to delete product", "error", err)
		return err
	}

	_ = s.pubsub.Publish(orgId, port.TopicProductDeleted, product)
	return nil
}

func (s *ProductService) CreateVariant(ctx context.Context, orgId string, productId string, input domain.CreateVariantInput) (domain.Variant, error) {
	variant, err := s.variantRepository.Create(ctx, domain.Variant{
		OrgId:       orgId,
		Id:          lib.GenerateId("var"),
		ProductId:   productId,
		Name:        input.Name,
		Description: input.Description,
		Metadata:    input.Metadata,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("failed to create variant", "error", err)
		return domain.Variant{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicVariantCreated, variant)
	return variant, nil
}

func (s *ProductService) GetVariant(ctx context.Context, orgId string, id string) (domain.Variant, error) {
	variant, err := s.variantRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find variant", "error", err)
		return domain.Variant{}, err
	}

	return variant, nil
}

func (s *ProductService) ListVariants(ctx context.Context, orgId string, productId string, pagination domain.Pagination) ([]domain.Variant, int, error) {
	variants, total, err := s.variantRepository.FindByProductId(ctx, orgId, productId, pagination)
	if err != nil {
		s.logger.Error("failed to list variants", "error", err)
		return nil, 0, err
	}

	return variants, total, nil
}

func (s *ProductService) UpdateVariant(ctx context.Context, orgId string, id string, input domain.UpdateVariantInput) (domain.Variant, error) {
	variant, err := s.variantRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find variant", "error", err)
		return domain.Variant{}, err
	}

	variant.Name = input.Name
	variant.Description = input.Description
	variant.Metadata = input.Metadata
	variant.UpdatedAt = time.Now().UTC()

	variant, err = s.variantRepository.Update(ctx, variant)
	if err != nil {
		s.logger.Error("failed to update variant", "error", err)
		return domain.Variant{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicVariantUpdated, variant)
	return variant, nil
}

func (s *ProductService) DeleteVariant(ctx context.Context, orgId string, id string) error {
	variant, err := s.variantRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find variant", "error", err)
		return err
	}

	err = s.variantRepository.Delete(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to delete variant", "error", err)
		return err
	}

	_ = s.pubsub.Publish(orgId, port.TopicVariantDeleted, variant)
	return nil
}

func (s *ProductService) GetPrice(ctx context.Context, orgId string, id string) (domain.Price, error) {
	price, err := s.priceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find price", "error", err)
		return domain.Price{}, err
	}

	return price, nil
}

func (s *ProductService) ListPrices(ctx context.Context, orgId string, variantId string, pagination domain.Pagination) ([]domain.Price, int, error) {
	prices, total, err := s.priceRepository.FindByVariantId(ctx, orgId, variantId, pagination)
	if err != nil {
		s.logger.Error("failed to list prices", "error", err)
		return nil, 0, err
	}

	return prices, total, nil
}

func (s *ProductService) UpdatePrice(ctx context.Context, orgId string, id string, input domain.CreatePriceInput) (domain.Price, error) {
	price, err := s.priceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find price", "error", err)
		return domain.Price{}, err
	}

	if input.BillingInterval == "" {
		input.BillingInterval = domain.BillingIntervalNone
	}
	if input.TrialInterval == "" {
		input.TrialInterval = domain.BillingIntervalNone
	}

	price.Label = input.Label
	price.Category = input.Category
	price.Scheme = input.Scheme
	price.Cycles = input.Cycles
	price.Currency = domain.Currency(input.Currency)
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

	price, err = s.priceRepository.Update(ctx, price)
	if err != nil {
		s.logger.Error("failed to update price", "error", err)
		return domain.Price{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicPriceUpdated, price)
	return price, nil
}

func (s *ProductService) DeletePrice(ctx context.Context, orgId string, id string) error {
	price, err := s.priceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to find price", "error", err)
		return err
	}

	err = s.priceRepository.Delete(ctx, orgId, id)
	if err != nil {
		s.logger.Error("failed to delete price", "error", err)
		return err
	}

	_ = s.pubsub.Publish(orgId, port.TopicPriceDeleted, price)
	return nil
}
