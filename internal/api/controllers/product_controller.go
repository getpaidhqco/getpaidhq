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

	product, err := s.productService.CreateProduct(c.Request.Context(), entities.CreateProductInput{
		OrgId: authUser.OrgId,
		Name:  input.Name,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, product)
}

// List all subscriptions
func (s ProductController) List(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	pagination := request.GetPagination(c)

	prods, err := s.productService.List(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.ListResponse{
		Data: prods,
		Meta: response.Meta{
			Total: len(prods),
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}
