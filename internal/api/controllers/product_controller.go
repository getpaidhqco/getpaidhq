package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/api/mappers"
	app_lib "payloop/internal/application/lib/authz"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

// ProductController data type
type ProductController struct {
	productService services.ProductService
	logger         logger.Logger
	authz          app_lib.Authz
}

// NewProductController creates new product controller
func NewProductController(productService services.ProductService, logger logger.Logger, authz app_lib.Authz) ProductController {
	return ProductController{
		productService: productService,
		logger:         logger,
		authz:          authz,
	}
}

func (s ProductController) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	id := c.Param("id")

	product, err := s.productService.FindById(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewProductFromEntity(product))
}

func (s ProductController) Create(c *gin.Context) {
	var apiInput request.CreateProductRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)

	allowed := s.authz.Enforce(authUser, app_lib.CreateProduct, "")
	if !allowed {
		apiErr := api.NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if err := c.ShouldBindJSON(&apiInput); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert API DTO to Application DTO
	appInput := mappers.ToCreateProductInput(apiInput)

	product, err := s.productService.CreateProduct(c.Request.Context(), authUser.OrgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewProductFromEntity(product))
}

// List all subscriptions
func (s ProductController) List(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	pagination := request.GetPagination(c)

	prods, total, err := s.productService.List(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	products := make([]response.Product, len(prods))
	for i, prod := range prods {
		products[i] = response.NewProductFromEntity(prod)
	}

	c.JSON(200, response.ListResponse{
		Data: products,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

func (s ProductController) Update(c *gin.Context) {
	var apiInput request.UpdateProductRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	id := c.Param("id")

	allowed := s.authz.Enforce(authUser, app_lib.UpdateProduct, "")
	if !allowed {
		apiErr := api.NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if err := c.ShouldBindJSON(&apiInput); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert API DTO to Application DTO
	appInput := mappers.ToUpdateProductInput(apiInput)

	product, err := s.productService.UpdateProduct(c.Request.Context(), orgId, id, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewProductFromEntity(product))
}

func (s ProductController) Delete(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	id := c.Param("id")

	allowed := s.authz.Enforce(authUser, app_lib.DeleteProduct, "")
	if !allowed {
		apiErr := api.NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	err := s.productService.DeleteProduct(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(204, nil)
}

// CreatePrice creates a new price for a variant
func (s ProductController) CreatePrice(c *gin.Context) {
	var input request.CreatePriceRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert request tiers to entity tiers
	var tiers []entities.CreatePriceTierInput
	for _, t := range input.Tiers {
		tiers = append(tiers, entities.CreatePriceTierInput{
			Tier:        t.Tier,
			FromQty:     t.FromQty,
			ToQty:       t.ToQty,
			UnitPrice:   t.UnitPrice,
			Description: t.Description,
		})
	}

	price, err := s.productService.CreateProductPrice(c.Request.Context(), entities.CreatePriceInput{
		OrgId:              orgId,
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

		// Usage-based billing fields
		HasUsage:           input.HasUsage,
		MeterId:            input.MeterId,
		PercentageRate:     input.PercentageRate,
		FixedFee:           input.FixedFee,
		OverageUnitPrice:   input.OverageUnitPrice,
		IncludedUsage:      input.IncludedUsage,
		UsageLimit:         input.UsageLimit,

		// Tier configuration
		Tiers:              tiers,

		Metadata:           input.Metadata,
	})
	if err != nil {
		s.logger.Error("Failed to create price", err.Error())
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, price)
}

// GetPrice gets a price by ID
func (s ProductController) GetPrice(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	price, err := s.productService.GetPrice(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, price)
}

// ListPrices lists all prices for a variant
func (s ProductController) ListPrices(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	variantId := c.Param("variantId")
	pagination := request.GetPagination(c)

	prices, total, err := s.productService.ListPrices(c.Request.Context(), orgId, variantId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.ListResponse{
		Data: prices,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// UpdatePrice updates a price
func (s ProductController) UpdatePrice(c *gin.Context) {
	var input request.CreatePriceRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("priceId")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	price, err := s.productService.UpdatePrice(c.Request.Context(), orgId, id, entities.CreatePriceInput{
		OrgId:              orgId,
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

		// Usage-based billing fields
		HasUsage:           input.HasUsage,
		MeterId:            input.MeterId,
		PercentageRate:     input.PercentageRate,
		FixedFee:           input.FixedFee,
		OverageUnitPrice:   input.OverageUnitPrice,
		IncludedUsage:      input.IncludedUsage,
		UsageLimit:         input.UsageLimit,

		Metadata:           input.Metadata,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, price)
}

// DeletePrice deletes a price
func (s ProductController) DeletePrice(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("priceId")

	err := s.productService.DeletePrice(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(204, nil)
}

// CreateVariant creates a new variant for a product
func (s ProductController) CreateVariant(c *gin.Context) {
	var input request.CreateVariantRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	productId := c.Param("productId")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	variant, err := s.productService.CreateVariant(c.Request.Context(), orgId, productId, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, variant)
}

// GetVariant gets a variant by ID
func (s ProductController) GetVariant(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	variant, err := s.productService.GetVariant(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, variant)
}

// ListVariants lists all variants for a product
func (s ProductController) ListVariants(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	productId := c.Param("productId")
	pagination := request.GetPagination(c)

	variants, total, err := s.productService.ListVariants(c.Request.Context(), orgId, productId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.ListResponse{
		Data: variants,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// UpdateVariant updates a variant
func (s ProductController) UpdateVariant(c *gin.Context) {
	var input request.UpdateVariantRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	variant, err := s.productService.UpdateVariant(c.Request.Context(), orgId, id, input)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, variant)
}

// DeleteVariant deletes a variant
func (s ProductController) DeleteVariant(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	err := s.productService.DeleteVariant(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(204, nil)
}
