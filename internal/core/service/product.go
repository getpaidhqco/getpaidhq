package service

import (
	"context"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
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

func (s *ProductService) CreateProduct(ctx context.Context, orgId string, input port.CreateProductInput) (domain.Product, error) {

	product, err := s.productRepository.Create(ctx,
		domain.Product{
			OrgId:       orgId,
			Id:          lib.GenerateId("prod"),
			Name:        input.Name,
			Description: input.Description,
			Status:      domain.ProductStatusActive,
			Metadata:    input.Metadata,
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		})
	if err != nil {
		s.logger.Error("Failed to create product", err.Error())
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
			s.logger.Error("Failed to create variant", err.Error())
			return domain.Product{}, err
		}

		for _, p := range v.Prices {
			_, err := s.priceRepository.Create(ctx,
				port.CreatePriceInput{
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
					BillableMetricId:   p.BillableMetricId,
					Tiers:              p.Tiers,
					FilterField:        p.FilterField,
					FilterValue:        p.FilterValue,
					Metadata:           p.Metadata,
				}.ToPrice(orgId, variant.Id))
			if err != nil {
				s.logger.Error("Failed to create price", err.Error())
				return domain.Product{}, err
			}
		}
	}

	product, err = s.FindById(ctx, orgId, product.Id)
	if err != nil {
		s.logger.Error("Failed to find product", err.Error())
		return domain.Product{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicProductCreated, product)
	return product, err
}

func (s *ProductService) List(ctx context.Context, orgId string, pagination domain.Pagination, statuses []domain.ProductStatus) ([]domain.Product, int, error) {
	subs, total, err := s.productRepository.Find(ctx, orgId, pagination, statuses)
	if err != nil {
		s.logger.Error("Failed to list products", err.Error())
		return nil, 0, err
	}

	return subs, total, nil
}

func (s *ProductService) FindById(ctx context.Context, orgId string, id string) (domain.Product, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to list products", err.Error())
		return domain.Product{}, err
	}

	return product, nil
}

func (s *ProductService) CreateProductPrice(ctx context.Context, input port.CreatePriceInput) (domain.Price, error) {
	if err := validatePriceConfig(input.Scheme, input.Tiers); err != nil {
		return domain.Price{}, err
	}

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
		BillableMetricId:   input.BillableMetricId,
		Tiers:              input.Tiers,
		FilterField:        input.FilterField,
		FilterValue:        input.FilterValue,
		Metadata:           input.Metadata,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	})

	if err != nil {
		s.logger.Error("Failed to create product price", err.Error())
		return domain.Price{}, err
	}

	_ = s.pubsub.Publish(input.OrgId, port.TopicPriceCreated, price)
	return price, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, orgId string, id string, input port.UpdateProductInput) (domain.Product, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find product", err.Error())
		return domain.Product{}, err
	}

	product.Name = input.Name
	product.Description = input.Description
	product.Metadata = input.Metadata
	product.UpdatedAt = time.Now().UTC()

	product, err = s.productRepository.Update(ctx, product)
	if err != nil {
		s.logger.Error("Failed to update product", err.Error())
		return domain.Product{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicProductUpdated, product)
	return product, nil
}

// ArchiveProduct retires a product: it is hidden from default listings and no
// longer sellable, but preserved in historic data. Idempotent — archiving an
// already-archived product returns it unchanged.
func (s *ProductService) ArchiveProduct(ctx context.Context, orgId string, id string) (domain.Product, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find product", err.Error())
		return domain.Product{}, err
	}
	if product.IsArchived() {
		return product, nil
	}

	now := time.Now().UTC()
	product.Status = domain.ProductStatusArchived
	product.ArchivedAt = &now
	product.UpdatedAt = now

	product, err = s.productRepository.Update(ctx, product)
	if err != nil {
		s.logger.Error("Failed to archive product", err.Error())
		return domain.Product{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicProductArchived, product)
	return product, nil
}

// UnarchiveProduct returns an archived product to active. Idempotent — an
// already-active product is returned unchanged.
func (s *ProductService) UnarchiveProduct(ctx context.Context, orgId string, id string) (domain.Product, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find product", err.Error())
		return domain.Product{}, err
	}
	if !product.IsArchived() {
		return product, nil
	}

	product.Status = domain.ProductStatusActive
	product.ArchivedAt = nil
	product.UpdatedAt = time.Now().UTC()

	product, err = s.productRepository.Update(ctx, product)
	if err != nil {
		s.logger.Error("Failed to unarchive product", err.Error())
		return domain.Product{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicProductUnarchived, product)
	return product, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, orgId string, id string) error {
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

	_ = s.pubsub.Publish(orgId, port.TopicProductDeleted, product)
	return nil
}

func (s *ProductService) CreateVariant(ctx context.Context, orgId string, productId string, input port.CreateVariantInput) (domain.Variant, error) {
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
		s.logger.Error("Failed to create variant", err.Error())
		return domain.Variant{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicVariantCreated, variant)
	return variant, nil
}

func (s *ProductService) GetVariant(ctx context.Context, orgId string, id string) (domain.Variant, error) {
	variant, err := s.variantRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find variant", err.Error())
		return domain.Variant{}, err
	}

	return variant, nil
}

func (s *ProductService) ListVariants(ctx context.Context, orgId string, productId string, pagination domain.Pagination) ([]domain.Variant, int, error) {
	variants, total, err := s.variantRepository.FindByProductId(ctx, orgId, productId, pagination)
	if err != nil {
		s.logger.Error("Failed to list variants", err.Error())
		return nil, 0, err
	}

	return variants, total, nil
}

func (s *ProductService) UpdateVariant(ctx context.Context, orgId string, id string, input port.UpdateVariantInput) (domain.Variant, error) {
	variant, err := s.variantRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find variant", err.Error())
		return domain.Variant{}, err
	}

	variant.Name = input.Name
	variant.Description = input.Description
	variant.Metadata = input.Metadata
	variant.UpdatedAt = time.Now().UTC()

	variant, err = s.variantRepository.Update(ctx, variant)
	if err != nil {
		s.logger.Error("Failed to update variant", err.Error())
		return domain.Variant{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicVariantUpdated, variant)
	return variant, nil
}

func (s *ProductService) DeleteVariant(ctx context.Context, orgId string, id string) error {
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

	_ = s.pubsub.Publish(orgId, port.TopicVariantDeleted, variant)
	return nil
}

func (s *ProductService) GetPrice(ctx context.Context, orgId string, id string) (domain.Price, error) {
	price, err := s.priceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find price", err.Error())
		return domain.Price{}, err
	}

	return price, nil
}

func (s *ProductService) ListPrices(ctx context.Context, orgId string, variantId string, pagination domain.Pagination) ([]domain.Price, int, error) {
	prices, total, err := s.priceRepository.FindByVariantId(ctx, orgId, variantId, pagination)
	if err != nil {
		s.logger.Error("Failed to list prices", err.Error())
		return nil, 0, err
	}

	return prices, total, nil
}

func (s *ProductService) UpdatePrice(ctx context.Context, orgId string, id string, input port.CreatePriceInput) (domain.Price, error) {
	if err := validatePriceConfig(input.Scheme, input.Tiers); err != nil {
		return domain.Price{}, err
	}
	price, err := s.priceRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to find price", err.Error())
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
	price.BillableMetricId = input.BillableMetricId
	price.Tiers = input.Tiers
	price.FilterField = input.FilterField
	price.FilterValue = input.FilterValue
	price.Metadata = input.Metadata
	price.UpdatedAt = time.Now().UTC()

	price, err = s.priceRepository.Update(ctx, price)
	if err != nil {
		s.logger.Error("Failed to update price", err.Error())
		return domain.Price{}, err
	}

	_ = s.pubsub.Publish(orgId, port.TopicPriceUpdated, price)
	return price, nil
}

func (s *ProductService) DeletePrice(ctx context.Context, orgId string, id string) error {
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

	_ = s.pubsub.Publish(orgId, port.TopicPriceDeleted, price)
	return nil
}

// GetDetails composes a ProductDetails read model: product + variants (each
// variant paired with its prices).
func (s *ProductService) GetDetails(ctx context.Context, orgId, id string) (ProductDetails, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		return ProductDetails{}, err
	}
	variants, _, err := s.variantRepository.FindByProductId(ctx, orgId, product.Id, domain.Pagination{Page: 1, Limit: 1000})
	if err != nil {
		return ProductDetails{}, err
	}
	variantIds := make([]string, 0, len(variants))
	for _, v := range variants {
		variantIds = append(variantIds, v.Id)
	}
	prices, err := s.priceRepository.FindByVariantIds(ctx, orgId, variantIds)
	if err != nil {
		return ProductDetails{}, err
	}
	pricesByVariant := make(map[string][]domain.Price, len(variants))
	for _, p := range prices {
		pricesByVariant[p.VariantId] = append(pricesByVariant[p.VariantId], p)
	}
	variantDetails := make([]VariantDetails, len(variants))
	for i, v := range variants {
		variantDetails[i] = VariantDetails{Variant: v, Prices: pricesByVariant[v.Id]}
	}
	return ProductDetails{Product: product, Variants: variantDetails}, nil
}

// ListDetails returns products with their composed details. Variants and
// prices are batch-loaded.
func (s *ProductService) ListDetails(ctx context.Context, orgId string, pagination domain.Pagination, statuses []domain.ProductStatus) ([]ProductDetails, int, error) {
	products, total, err := s.productRepository.Find(ctx, orgId, pagination, statuses)
	if err != nil {
		return nil, 0, err
	}
	if len(products) == 0 {
		return []ProductDetails{}, total, nil
	}
	out := make([]ProductDetails, len(products))
	for i, p := range products {
		variants, _, err := s.variantRepository.FindByProductId(ctx, orgId, p.Id, domain.Pagination{Page: 1, Limit: 1000})
		if err != nil {
			return nil, 0, err
		}
		variantIds := make([]string, 0, len(variants))
		for _, v := range variants {
			variantIds = append(variantIds, v.Id)
		}
		prices, err := s.priceRepository.FindByVariantIds(ctx, orgId, variantIds)
		if err != nil {
			return nil, 0, err
		}
		pricesByVariant := make(map[string][]domain.Price, len(variants))
		for _, pr := range prices {
			pricesByVariant[pr.VariantId] = append(pricesByVariant[pr.VariantId], pr)
		}
		variantDetails := make([]VariantDetails, len(variants))
		for j, v := range variants {
			variantDetails[j] = VariantDetails{Variant: v, Prices: pricesByVariant[v.Id]}
		}
		out[i] = ProductDetails{Product: p, Variants: variantDetails}
	}
	return out, total, nil
}

// GetVariantDetails composes a VariantDetails read model.
func (s *ProductService) GetVariantDetails(ctx context.Context, orgId, id string) (VariantDetails, error) {
	variant, err := s.variantRepository.FindById(ctx, orgId, id)
	if err != nil {
		return VariantDetails{}, err
	}
	prices, _, err := s.priceRepository.FindByVariantId(ctx, orgId, variant.Id, domain.Pagination{Page: 1, Limit: 1000})
	if err != nil {
		return VariantDetails{}, err
	}
	return VariantDetails{Variant: variant, Prices: prices}, nil
}

// validatePriceConfig checks a graduated/volume/tiered scheme carries at least one
// rate tier. (Metering needs no category check — a price is metered iff it has a
// meter attached; see Price.IsMetered.)
func validatePriceConfig(scheme domain.PriceScheme, tiers []domain.PriceTier) error {
	switch scheme {
	case domain.Graduated, domain.Volume, domain.Tiered:
		if len(tiers) == 0 {
			return lib.NewCustomError(lib.BadRequestError, "tiers are required for graduated, volume, or tiered schemes", nil)
		}
	}
	return nil
}
