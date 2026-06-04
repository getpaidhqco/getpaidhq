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
	fuego.Get(products, "", s.List, option.Summary("List products"))
	fuego.Get(products, "/{id}", s.Get, option.Summary("Get a product"))
	fuego.Post(products, "", s.Create, option.Summary("Create a product"))
	fuego.Patch(products, "/{id}", s.Update, option.Summary("Update a product"))
	fuego.Delete(products, "/{id}", s.Delete, option.Summary("Delete a product"))
	fuego.Get(products, "/{id}/variants", s.ListVariants, option.Summary("List variants of a product"))
	fuego.Post(products, "/{id}/variants", s.CreateVariant, option.Summary("Add a variant to a product"))

	variants := fuego.Group(srv, "/variants", option.Tags("Variants"))
	fuego.Get(variants, "/{variantId}", s.GetVariant, option.Summary("Get a variant"))
	fuego.Put(variants, "/{variantId}", s.UpdateVariant, option.Summary("Update a variant"))
	fuego.Delete(variants, "/{variantId}", s.DeleteVariant, option.Summary("Delete a variant"))
	fuego.Get(variants, "/{variantId}/prices", s.ListPrices, option.Summary("List prices of a variant"))

	prices := fuego.Group(srv, "/prices", option.Tags("Prices"))
	fuego.Get(prices, "/{priceId}", s.GetPrice, option.Summary("Get a price"))
	fuego.Post(prices, "", s.CreatePrice, option.Summary("Create a price"))
	fuego.Patch(prices, "/{priceId}", s.UpdatePrice, option.Summary("Update a price"))
	fuego.Delete(prices, "/{priceId}", s.DeletePrice, option.Summary("Delete a price"))
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
	product, err := s.productService.FindById(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	return NewProductFromEntity(product), nil
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
	variants := make([]service.CreateProductVariantInput, len(req.Variants))
	for i, v := range req.Variants {
		prices := make([]service.CreateProductPriceInput, len(v.Prices))
		for j, p := range v.Prices {
			prices[j] = service.CreateProductPriceInput{
				Label:              p.Label,
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
			}
		}
		variants[i] = service.CreateProductVariantInput{
			Name:        v.Name,
			Description: v.Description,
			Metadata:    v.Metadata,
			Prices:      prices,
		}
	}
	input := service.CreateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
		Variants:    variants,
	}
	product, err := s.productService.CreateProduct(c.Context(), authUser.OrgId, input)
	if err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	return NewProductFromEntity(product), nil
}

func (s *ProductHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, s.authz, port.ActionListProducts); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	pagination := GetPagination(c)
	prods, total, err := s.productService.List(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	products := make([]ProductResponse, len(prods))
	for i, prod := range prods {
		products[i] = NewProductFromEntity(prod)
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
	input := service.UpdateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
	product, err := s.productService.UpdateProduct(c.Context(), authUser.OrgId, c.PathParam("id"), input)
	if err != nil {
		return ProductResponse{}, NewApiErrorFromError(err)
	}
	return NewProductFromEntity(product), nil
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

func (s *ProductHandler) CreatePrice(c fuego.ContextWithBody[CreatePriceRequest]) (domain.Price, error) {
	if err := enforce(c, s.authz, port.ActionCreatePrice); err != nil {
		return domain.Price{}, err
	}
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return domain.Price{}, err
	}
	price, err := s.productService.CreateProductPrice(c.Context(), service.CreatePriceInput{
		OrgId:              authUser.OrgId,
		VariantId:          input.VariantId,
		Category:           input.Category,
		Label:              input.Label,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Currency:           input.Currency,
		UnitPrice:          input.UnitPrice,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		Metadata:           input.Metadata,
	})
	if err != nil {
		// Single-handling rule: this error is returned to the caller; logging
		// it here would produce a duplicate entry in aggregators when the
		// serializer / request-logging middleware records the response.
		return domain.Price{}, NewApiErrorFromError(err)
	}
	return price, nil
}

func (s *ProductHandler) GetPrice(c fuego.ContextNoBody) (domain.Price, error) {
	if err := enforce(c, s.authz, port.ActionGetPrice); err != nil {
		return domain.Price{}, err
	}
	authUser := AuthUserFrom(c)
	price, err := s.productService.GetPrice(c.Context(), authUser.OrgId, c.PathParam("priceId"))
	if err != nil {
		return domain.Price{}, NewApiErrorFromError(err)
	}
	return price, nil
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

func (s *ProductHandler) UpdatePrice(c fuego.ContextWithBody[CreatePriceRequest]) (domain.Price, error) {
	if err := enforce(c, s.authz, port.ActionUpdatePrice); err != nil {
		return domain.Price{}, err
	}
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return domain.Price{}, err
	}
	price, err := s.productService.UpdatePrice(c.Context(), authUser.OrgId, c.PathParam("priceId"), service.CreatePriceInput{
		OrgId:              authUser.OrgId,
		VariantId:          input.VariantId,
		Category:           input.Category,
		Scheme:             input.Scheme,
		Cycles:             input.Cycles,
		Label:              input.Label,
		Currency:           input.Currency,
		UnitPrice:          input.UnitPrice,
		MinPrice:           input.MinPrice,
		SuggestedPrice:     input.SuggestedPrice,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		TaxCode:            input.TaxCode,
		Metadata:           input.Metadata,
	})
	if err != nil {
		return domain.Price{}, NewApiErrorFromError(err)
	}
	return price, nil
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

func (s *ProductHandler) CreateVariant(c fuego.ContextWithBody[CreateVariantRequest]) (domain.Variant, error) {
	if err := enforce(c, s.authz, port.ActionCreateVariant); err != nil {
		return domain.Variant{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return domain.Variant{}, err
	}
	input := service.CreateVariantInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
	variant, err := s.productService.CreateVariant(c.Context(), authUser.OrgId, c.PathParam("id"), input)
	if err != nil {
		return domain.Variant{}, NewApiErrorFromError(err)
	}
	return variant, nil
}

func (s *ProductHandler) GetVariant(c fuego.ContextNoBody) (domain.Variant, error) {
	if err := enforce(c, s.authz, port.ActionGetVariant); err != nil {
		return domain.Variant{}, err
	}
	authUser := AuthUserFrom(c)
	variant, err := s.productService.GetVariant(c.Context(), authUser.OrgId, c.PathParam("variantId"))
	if err != nil {
		return domain.Variant{}, NewApiErrorFromError(err)
	}
	return variant, nil
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

func (s *ProductHandler) UpdateVariant(c fuego.ContextWithBody[UpdateVariantRequest]) (domain.Variant, error) {
	if err := enforce(c, s.authz, port.ActionUpdateVariant); err != nil {
		return domain.Variant{}, err
	}
	authUser := AuthUserFrom(c)
	req, err := c.Body()
	if err != nil {
		return domain.Variant{}, err
	}
	input := service.UpdateVariantInput{
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
	}
	variant, err := s.productService.UpdateVariant(c.Context(), authUser.OrgId, c.PathParam("variantId"), input)
	if err != nil {
		return domain.Variant{}, NewApiErrorFromError(err)
	}
	return variant, nil
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
