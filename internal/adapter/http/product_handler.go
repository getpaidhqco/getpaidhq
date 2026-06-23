package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// ProductHandler handles HTTP requests for products, variants, and prices.
type ProductHandler struct {
	productService *service.ProductService
	logger         port.Logger
	authz          port.Authz
}

func NewProductHandler(productService *service.ProductService, logger port.Logger, authz port.Authz) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		logger:         logger,
		authz:          authz,
	}
}

func (s *ProductHandler) RegisterRoutes(srv *fuego.Server) {
	products := fuego.Group(srv, "/products", option.Tags("Products"))
	fuego.Get(products, "", s.List, append(PaginationParams(), option.Summary("List products"), option.OperationID("listProducts"))...)
	fuego.Get(products, "/{id}", s.Get, option.Summary("Get a product"), option.OperationID("getProduct"))
	fuego.Post(products, "", s.Create, option.Summary("Create a product"), option.OperationID("createProduct"))
	fuego.Patch(products, "/{id}", s.Update, option.Summary("Update a product"), option.OperationID("updateProduct"))
	fuego.Delete(products, "/{id}", s.Delete, option.Summary("Delete a product"), option.OperationID("deleteProduct"))
	fuego.Post(products, "/{id}/archive", s.Archive, option.Summary("Archive a product"), option.OperationID("archiveProduct"))
	fuego.Post(products, "/{id}/unarchive", s.Unarchive, option.Summary("Unarchive a product"), option.OperationID("unarchiveProduct"))
	fuego.Get(products, "/{id}/variants", s.ListVariants, append(PaginationParams(), option.Summary("List variants of a product"), option.OperationID("listProductVariants"))...)
	fuego.Post(products, "/{id}/variants", s.CreateVariant, option.Summary("Add a variant to a product"), option.OperationID("createProductVariant"))

	variants := fuego.Group(srv, "/variants", option.Tags("Variants"))
	fuego.Get(variants, "/{variantId}", s.GetVariant, option.Summary("Get a variant"), option.OperationID("getVariant"))
	fuego.Put(variants, "/{variantId}", s.UpdateVariant, option.Summary("Update a variant"), option.OperationID("updateVariant"))
	fuego.Delete(variants, "/{variantId}", s.DeleteVariant, option.Summary("Delete a variant"), option.OperationID("deleteVariant"))
	fuego.Get(variants, "/{variantId}/prices", s.ListPrices, append(PaginationParams(), option.Summary("List prices of a variant"), option.OperationID("listVariantPrices"))...)

	prices := fuego.Group(srv, "/prices", option.Tags("Prices"))
	fuego.Get(prices, "/{priceId}", s.GetPrice, option.Summary("Get a price"), option.OperationID("getPrice"))
	fuego.Post(prices, "", s.CreatePrice, option.Summary("Create a price"), option.OperationID("createPrice"))
	fuego.Patch(prices, "/{priceId}", s.UpdatePrice, option.Summary("Update a price"), option.OperationID("updatePrice"))
	fuego.Delete(prices, "/{priceId}", s.DeletePrice, option.Summary("Delete a price"), option.OperationID("deletePrice"))
}

func enforce[B, P any](c fuego.Context[B, P], authz port.Authz, action port.Action) error {
	if !authz.Enforce(AuthUserFrom(c), action, "") {
		return NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	return nil
}

func (s *ProductHandler) Get(c fuego.ContextNoBody) (ProductResponse, error) {
	if err := enforce(c, s.authz, port.ActionGetProduct); err != nil {
		return ProductResponse{}, err
	}
	authUser := AuthUserFrom(c)
	details, err := s.productService.GetDetails(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	return NewProductResponseFromDetails(details), nil
}

func (s *ProductHandler) Create(c fuego.ContextWithBody[CreateProductRequest]) (ProductResponse, error) {
	if err := enforce(c, s.authz, port.ActionCreateProduct); err != nil {
		return ProductResponse{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return ProductResponse{}, err
	}
	variants := make([]port.CreateProductVariantInput, len(req.Variants))
	for i, v := range req.Variants {
		prices := make([]port.CreateProductPriceInput, len(v.Prices))
		for j, p := range v.Prices {
			tiers, terr := toDomainTiers(p.Tiers)
			if terr != nil {
				return ProductResponse{}, NewApiError(lib.BadRequestError, "invalid tier value", terr)
			}
			prices[j] = port.CreateProductPriceInput{
				Label:              p.Label,
				Category:           p.Category,
				Scheme:             p.Scheme,
				Cycles:             p.Cycles,
				Currency:           p.Currency,
				UnitPrice:          p.UnitPrice,
				UnitCount:          p.UnitCount,
				MinPrice:           p.MinPrice,
				SuggestedPrice:     p.SuggestedPrice,
				BillingInterval:    p.BillingInterval,
				BillingIntervalQty: p.BillingIntervalQty,
				TrialInterval:      p.TrialInterval,
				TrialIntervalQty:   p.TrialIntervalQty,
				TaxCode:            p.TaxCode,
				BillableMetricId:   p.BillableMetricId,
				Tiers:              tiers,
				FilterField:        p.FilterField,
				FilterValue:        p.FilterValue,
				ProrateOnIncrease:  p.ProrateOnIncrease,
				CreditOnDecrease:   p.CreditOnDecrease,
				Metadata:           p.Metadata,
			}
		}
		variants[i] = port.CreateProductVariantInput{
			Name:        v.Name,
			Description: v.Description,
			Metadata:    v.Metadata,
			Prices:      prices,
		}
	}
	input := port.CreateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
		Variants:    variants,
	}
	product, err := s.productService.CreateProduct(c.Context(), authUser.OrgId, input)
	if err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	details, err := s.productService.GetDetails(c.Context(), authUser.OrgId, product.Id)
	if err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	return NewProductResponseFromDetails(details), nil
}

// parseProductStatusFilter maps the ?status= query param to the status filter
// passed to the service. Default (absent or "active") lists only active
// products — archived products are hidden from the dashboard. "all" disables the
// filter; any other value is rejected as a 400.
func parseProductStatusFilter[B, P any](c fuego.Context[B, P]) ([]domain.ProductStatus, error) {
	switch c.QueryParam("status") {
	case "", string(domain.ProductStatusActive):
		return []domain.ProductStatus{domain.ProductStatusActive}, nil
	case string(domain.ProductStatusArchived):
		return []domain.ProductStatus{domain.ProductStatusArchived}, nil
	case "all":
		return nil, nil
	default:
		return nil, NewApiError(lib.BadRequestError, "status must be one of: active, archived, all", nil)
	}
}

func (s *ProductHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, s.authz, port.ActionListProducts); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	pagination := GetPagination(c)
	statuses, err := parseProductStatusFilter(c)
	if err != nil {
		return ListResponse{}, err
	}
	details, total, err := s.productService.ListDetails(c.Context(), authUser.OrgId, pagination, statuses)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	products := make([]ProductResponse, len(details))
	for i, d := range details {
		products[i] = NewProductResponseFromDetails(d)
	}
	return ListResponse{
		Data: products,
		Meta: Meta{Total: total, Page: pagination.Page, Limit: pagination.Limit},
	}, nil
}

func (s *ProductHandler) Update(c fuego.ContextWithBody[UpdateProductRequest]) (ProductResponse, error) {
	if err := enforce(c, s.authz, port.ActionUpdateProduct); err != nil {
		return ProductResponse{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return ProductResponse{}, err
	}
	input := port.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
	if _, err := s.productService.UpdateProduct(c.Context(), authUser.OrgId, c.PathParam("id"), input); err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	details, err := s.productService.GetDetails(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	return NewProductResponseFromDetails(details), nil
}

func (s *ProductHandler) Delete(c fuego.ContextNoBody) (EmptyResponse, error) {
	if err := enforce(c, s.authz, port.ActionDeleteProduct); err != nil {
		return EmptyResponse{}, err
	}
	authUser := AuthUserFrom(c)
	if err := s.productService.DeleteProduct(c.Context(), authUser.OrgId, c.PathParam("id")); err != nil {
		return EmptyResponse{}, NewApiErrorFromError(err)
	}
	c.SetStatus(204)
	return EmptyResponse{}, nil
}

// Archive retires a product (hidden from default listings, not sellable). Reuses
// the UpdateProduct permission — archiving is a privileged product mutation.
func (s *ProductHandler) Archive(c fuego.ContextNoBody) (ProductResponse, error) {
	if err := enforce(c, s.authz, port.ActionUpdateProduct); err != nil {
		return ProductResponse{}, err
	}
	authUser := AuthUserFrom(c)
	if _, err := s.productService.ArchiveProduct(c.Context(), authUser.OrgId, c.PathParam("id")); err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	details, err := s.productService.GetDetails(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	return NewProductResponseFromDetails(details), nil
}

// Unarchive returns an archived product to active.
func (s *ProductHandler) Unarchive(c fuego.ContextNoBody) (ProductResponse, error) {
	if err := enforce(c, s.authz, port.ActionUpdateProduct); err != nil {
		return ProductResponse{}, err
	}
	authUser := AuthUserFrom(c)
	if _, err := s.productService.UnarchiveProduct(c.Context(), authUser.OrgId, c.PathParam("id")); err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	details, err := s.productService.GetDetails(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	return NewProductResponseFromDetails(details), nil
}

func (s *ProductHandler) CreatePrice(c fuego.ContextWithBody[CreatePriceRequest]) (PriceResponse, error) {
	if err := enforce(c, s.authz, port.ActionCreatePrice); err != nil {
		return PriceResponse{}, err
	}
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return PriceResponse{}, err
	}
	tiers, err := toDomainTiers(input.Tiers)
	if err != nil {
		return PriceResponse{}, NewApiError(lib.BadRequestError, "invalid tier value", err)
	}
	price, err := s.productService.CreateProductPrice(c.Context(), port.CreatePriceInput{
		OrgId:              authUser.OrgId,
		VariantId:          input.VariantId,
		Category:           input.Category,
		Label:              input.Label,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           input.Currency,
		UnitPrice:          input.UnitPrice,
		UnitCount:          input.UnitCount,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		BillableMetricId:   input.BillableMetricId,
		Tiers:              tiers,
		FilterField:        input.FilterField,
		FilterValue:        input.FilterValue,
		ProrateOnIncrease:  input.ProrateOnIncrease,
		CreditOnDecrease:   input.CreditOnDecrease,
		Metadata:           input.Metadata,
	})
	if err != nil {
		// Single-handling rule: this error is returned to the caller; logging
		// it here would produce a duplicate entry in aggregators when the
		// serializer / request-logging middleware records the response.
		return PriceResponse{}, NewApiErrorFromError(err)
	}
	return NewPriceFromEntity(price), nil
}

func (s *ProductHandler) GetPrice(c fuego.ContextNoBody) (PriceResponse, error) {
	if err := enforce(c, s.authz, port.ActionGetPrice); err != nil {
		return PriceResponse{}, err
	}
	authUser := AuthUserFrom(c)
	price, err := s.productService.GetPrice(c.Context(), authUser.OrgId, c.PathParam("priceId"))
	if err != nil {
		return PriceResponse{}, NewApiErrorFromError(err)
	}
	return NewPriceFromEntity(price), nil
}

func (s *ProductHandler) ListPrices(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, s.authz, port.ActionListPrices); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	pagination := GetPagination(c)
	prices, total, err := s.productService.ListPrices(c.Context(), authUser.OrgId, c.PathParam("variantId"), pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	return ListResponse{
		Data: prices,
		Meta: Meta{Total: total, Page: pagination.Page, Limit: pagination.Limit},
	}, nil
}

func (s *ProductHandler) UpdatePrice(c fuego.ContextWithBody[CreatePriceRequest]) (PriceResponse, error) {
	if err := enforce(c, s.authz, port.ActionUpdatePrice); err != nil {
		return PriceResponse{}, err
	}
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return PriceResponse{}, err
	}
	tiers, err := toDomainTiers(input.Tiers)
	if err != nil {
		return PriceResponse{}, NewApiError(lib.BadRequestError, "invalid tier value", err)
	}
	price, err := s.productService.UpdatePrice(c.Context(), authUser.OrgId, c.PathParam("priceId"), port.CreatePriceInput{
		OrgId:              authUser.OrgId,
		VariantId:          input.VariantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Label:              input.Label,
		Currency:           input.Currency,
		UnitPrice:          input.UnitPrice,
		UnitCount:          input.UnitCount,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		BillableMetricId:   input.BillableMetricId,
		Tiers:              tiers,
		FilterField:        input.FilterField,
		FilterValue:        input.FilterValue,
		ProrateOnIncrease:  input.ProrateOnIncrease,
		CreditOnDecrease:   input.CreditOnDecrease,
		Metadata:           input.Metadata,
	})
	if err != nil {
		return PriceResponse{}, NewApiErrorFromError(err)
	}
	return NewPriceFromEntity(price), nil
}

func (s *ProductHandler) DeletePrice(c fuego.ContextNoBody) (EmptyResponse, error) {
	if err := enforce(c, s.authz, port.ActionDeletePrice); err != nil {
		return EmptyResponse{}, err
	}
	authUser := AuthUserFrom(c)
	if err := s.productService.DeletePrice(c.Context(), authUser.OrgId, c.PathParam("priceId")); err != nil {
		return EmptyResponse{}, NewApiErrorFromError(err)
	}
	c.SetStatus(204)
	return EmptyResponse{}, nil
}

func (s *ProductHandler) CreateVariant(c fuego.ContextWithBody[CreateVariantRequest]) (VariantResponse, error) {
	if err := enforce(c, s.authz, port.ActionCreateVariant); err != nil {
		return VariantResponse{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return VariantResponse{}, err
	}
	input := port.CreateVariantInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
	variant, err := s.productService.CreateVariant(c.Context(), authUser.OrgId, c.PathParam("id"), input)
	if err != nil {
		return VariantResponse{}, NewApiErrorFromError(err)
	}
	return NewVariantResponseFromDetails(service.VariantDetails{Variant: variant}), nil
}

func (s *ProductHandler) GetVariant(c fuego.ContextNoBody) (VariantResponse, error) {
	if err := enforce(c, s.authz, port.ActionGetVariant); err != nil {
		return VariantResponse{}, err
	}
	authUser := AuthUserFrom(c)
	variant, err := s.productService.GetVariant(c.Context(), authUser.OrgId, c.PathParam("variantId"))
	if err != nil {
		return VariantResponse{}, NewApiErrorFromError(err)
	}
	return NewVariantResponseFromDetails(service.VariantDetails{Variant: variant}), nil
}

func (s *ProductHandler) ListVariants(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, s.authz, port.ActionListVariants); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	pagination := GetPagination(c)
	variants, total, err := s.productService.ListVariants(c.Context(), authUser.OrgId, c.PathParam("id"), pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	return ListResponse{
		Data: variants,
		Meta: Meta{Total: total, Page: pagination.Page, Limit: pagination.Limit},
	}, nil
}

func (s *ProductHandler) UpdateVariant(c fuego.ContextWithBody[UpdateVariantRequest]) (VariantResponse, error) {
	if err := enforce(c, s.authz, port.ActionUpdateVariant); err != nil {
		return VariantResponse{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return VariantResponse{}, err
	}
	input := port.UpdateVariantInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
	variant, err := s.productService.UpdateVariant(c.Context(), authUser.OrgId, c.PathParam("variantId"), input)
	if err != nil {
		return VariantResponse{}, NewApiErrorFromError(err)
	}
	return NewVariantResponseFromDetails(service.VariantDetails{Variant: variant}), nil
}

func (s *ProductHandler) DeleteVariant(c fuego.ContextNoBody) (EmptyResponse, error) {
	if err := enforce(c, s.authz, port.ActionDeleteVariant); err != nil {
		return EmptyResponse{}, err
	}
	authUser := AuthUserFrom(c)
	if err := s.productService.DeleteVariant(c.Context(), authUser.OrgId, c.PathParam("variantId")); err != nil {
		return EmptyResponse{}, NewApiErrorFromError(err)
	}
	c.SetStatus(204)
	return EmptyResponse{}, nil
}
