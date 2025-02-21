package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
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
	var input request.CreateProductRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)

	allowed := s.authz.Enforce(authUser, app_lib.CreateProduct, "")
	if !allowed {
		apiErr := api.NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	product, err := s.productService.CreateProduct(c.Request.Context(), authUser.OrgId, input)
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

// List all subscriptions
func (s ProductController) CreatePrice(c *gin.Context) {
	var input request.CreatePriceRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	price, err := s.productService.CreateProductPrice(c.Request.Context(), entities.CreatePriceInput{
		OrgId:              orgId,
		VariantId:          input.VariantId,
		Category:           input.Category,
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
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, price)
}
