package handler

import (
	"github.com/gin-gonic/gin"

	"payloop/internal/application/services"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

// CartHandler handles HTTP requests for carts.
type CartHandler struct {
	cartService services.CartService
	logger      port.Logger
}

// NewCartHandler creates a new CartHandler.
func NewCartHandler(cartService services.CartService, logger port.Logger) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		logger:      logger,
	}
}

// RegisterRoutes registers cart routes on the given router group.
func (o *CartHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/carts/:id/add", o.AddProduct)
	rg.POST("/carts/:id/remove", o.RemoveItem)
}

func (o *CartHandler) AddProduct(c *gin.Context) {
	var input AddItemRequest
	cartId := c.Param("id")
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	qty := input.Quantity
	if qty <= 0 {
		qty = 1
	}
	_ = qty // original code assigned but only used AddProductCommand.Quantity

	cart, err := o.cartService.AddProduct(c.Request.Context(), domain.AddProductCommand{
		OrgId:     orgId,
		CartId:    cartId,
		ProductId: input.ProductId,
		PriceId:   input.PriceId,
		Quantity:  input.Quantity,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	response := ToCartResponse(cart)
	c.JSON(200, response)
}

func (o *CartHandler) RemoveItem(c *gin.Context) {
	var input RemoveItemRequest
	cartId := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	cart, err := o.cartService.RemoveItem(c.Request.Context(), domain.RemoveItemCommand{
		OrgId:  input.OrgId,
		CartId: cartId,
		Id:     input.Id,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	response := ToCartResponse(cart)
	c.JSON(200, response)
}
