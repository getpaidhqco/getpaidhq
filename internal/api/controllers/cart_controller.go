package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/mapper"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities/carts"
	"payloop/internal/lib"
)

var validate *validator.Validate

// CartController data type
type CartController struct {
	cartService services.CartService
	logger      lib.Logger
}

// NewCartController creates new Cart controller
func NewCartController(cartService services.CartService, logger lib.Logger) CartController {
	return CartController{
		cartService: cartService,
		logger:      logger,
	}
}

func (o *CartController) AddProduct(c *gin.Context) {
	var input request.AddItemRequest
	cartId := c.Param("id")
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	qty := input.Quantity
	if qty <= 0 {
		qty = 1
	}

	cart, err := o.cartService.AddProduct(c.Request.Context(), carts.AddProductCommand{
		OrgId:     orgId,
		CartId:    cartId,
		ProductId: input.ProductId,
		PriceId:   input.PriceId,
		Quantity:  input.Quantity,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	response := mapper.ToCartResponse(cart)

	c.JSON(200, response)
}

func (o *CartController) RemoveItem(c *gin.Context) {
	var input request.RemoveItemRequest
	cartId := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	cart, err := o.cartService.RemoveItem(c.Request.Context(), carts.RemoveItemCommand{
		OrgId:  input.OrgId,
		CartId: cartId,
		Id:     input.Id,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	response := mapper.ToCartResponse(cart)

	c.JSON(200, response)
}
