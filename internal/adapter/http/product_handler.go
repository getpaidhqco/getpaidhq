package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"

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

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(productService *service.ProductService, logger port.Logger, authz port.Authz) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		logger:         logger,
		authz:          authz,
	}
}

// RegisterRoutes registers product, variant, and price routes on the given router group.
func (s *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Product routes
	rg.GET("/products", s.checkAuthz(port.ActionListProducts), s.List)
	rg.GET("/products/:id", s.checkAuthz(port.ActionGetProduct), s.Get)
	rg.POST("/products", s.checkAuthz(port.ActionCreateProduct), s.Create)
	rg.PATCH("/products/:id", s.checkAuthz(port.ActionUpdateProduct), s.Update)
	rg.DELETE("/products/:id", s.checkAuthz(port.ActionDeleteProduct), s.Delete)

	// Variant routes
	rg.GET("/variants/:variantId", s.checkAuthz(port.ActionGetVariant), s.GetVariant)
	rg.GET("/products/:id/variants", s.checkAuthz(port.ActionListVariants), s.ListVariants)
	rg.POST("/products/:id/variants", s.checkAuthz(port.ActionCreateVariant), s.CreateVariant)
	rg.PUT("/variants/:variantId", s.checkAuthz(port.ActionUpdateVariant), s.UpdateVariant)
	rg.DELETE("/variants/:variantId", s.checkAuthz(port.ActionDeleteVariant), s.DeleteVariant)

	// Price routes
	rg.GET("/prices/:priceId", s.checkAuthz(port.ActionGetPrice), s.GetPrice)
	rg.GET("/variants/:variantId/prices", s.checkAuthz(port.ActionListPrices), s.ListPrices)
	rg.POST("/prices", s.checkAuthz(port.ActionCreatePrice), s.CreatePrice)
	rg.PATCH("/prices/:priceId", s.checkAuthz(port.ActionUpdatePrice), s.UpdatePrice)
	rg.DELETE("/prices/:priceId", s.checkAuthz(port.ActionDeletePrice), s.DeletePrice)
}

func (s *ProductHandler) checkAuthz(action port.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		authUser := user.(port.AuthUser)
		allowed := s.authz.Enforce(authUser, action, "")
		if !allowed {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *ProductHandler) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	orgId := authUser.OrgId
	id := c.Param("id")

	product, err := s.productService.FindById(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, NewProductFromEntity(product))
}

func (s *ProductHandler) Create(c *gin.Context) {
	var input domain.CreateProductInput
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)

	allowed := s.authz.Enforce(authUser, port.ActionCreateProduct, "")
	if !allowed {
		apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	product, err := s.productService.CreateProduct(c.Request.Context(), authUser.OrgId, input)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, NewProductFromEntity(product))
}

func (s *ProductHandler) List(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	pagination := GetPagination(c)

	prods, total, err := s.productService.List(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	products := make([]ProductResponse, len(prods))
	for i, prod := range prods {
		products[i] = NewProductFromEntity(prod)
	}

	c.JSON(200, ListResponse{
		Data: products,
		Meta: Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

func (s *ProductHandler) Update(c *gin.Context) {
	var input domain.UpdateProductInput
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	orgId := authUser.OrgId
	id := c.Param("id")

	allowed := s.authz.Enforce(authUser, port.ActionUpdateProduct, "")
	if !allowed {
		apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	product, err := s.productService.UpdateProduct(c.Request.Context(), orgId, id, input)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, NewProductFromEntity(product))
}

func (s *ProductHandler) Delete(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	orgId := authUser.OrgId
	id := c.Param("id")

	allowed := s.authz.Enforce(authUser, port.ActionDeleteProduct, "")
	if !allowed {
		apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	err := s.productService.DeleteProduct(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(204, nil)
}

// CreatePrice creates a new price for a variant.
func (s *ProductHandler) CreatePrice(c *gin.Context) {
	var input CreatePriceRequest
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	price, err := s.productService.CreateProductPrice(c.Request.Context(), domain.CreatePriceInput{
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
		Metadata:           input.Metadata,
	})
	if err != nil {
		s.logger.Error("Failed to create price", err.Error())
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, price)
}

// GetPrice gets a price by ID.
func (s *ProductHandler) GetPrice(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("id")

	price, err := s.productService.GetPrice(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, price)
}

// ListPrices lists all prices for a variant.
func (s *ProductHandler) ListPrices(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	variantId := c.Param("variantId")
	pagination := GetPagination(c)

	prices, total, err := s.productService.ListPrices(c.Request.Context(), orgId, variantId, pagination)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, ListResponse{
		Data: prices,
		Meta: Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// UpdatePrice updates a price.
func (s *ProductHandler) UpdatePrice(c *gin.Context) {
	var input CreatePriceRequest
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("priceId")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	price, err := s.productService.UpdatePrice(c.Request.Context(), orgId, id, domain.CreatePriceInput{
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
		Metadata:           input.Metadata,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, price)
}

// DeletePrice deletes a price.
func (s *ProductHandler) DeletePrice(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("priceId")

	err := s.productService.DeletePrice(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(204, nil)
}

// CreateVariant creates a new variant for a product.
func (s *ProductHandler) CreateVariant(c *gin.Context) {
	var input domain.CreateVariantInput
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	productId := c.Param("productId")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	variant, err := s.productService.CreateVariant(c.Request.Context(), orgId, productId, input)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, variant)
}

// GetVariant gets a variant by ID.
func (s *ProductHandler) GetVariant(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("id")

	variant, err := s.productService.GetVariant(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, variant)
}

// ListVariants lists all variants for a product.
func (s *ProductHandler) ListVariants(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	productId := c.Param("productId")
	pagination := GetPagination(c)

	variants, total, err := s.productService.ListVariants(c.Request.Context(), orgId, productId, pagination)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, ListResponse{
		Data: variants,
		Meta: Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// UpdateVariant updates a variant.
func (s *ProductHandler) UpdateVariant(c *gin.Context) {
	var input domain.UpdateVariantInput
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	variant, err := s.productService.UpdateVariant(c.Request.Context(), orgId, id, input)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, variant)
}

// DeleteVariant deletes a variant.
func (s *ProductHandler) DeleteVariant(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("id")

	err := s.productService.DeleteVariant(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(204, nil)
}
