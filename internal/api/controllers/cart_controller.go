package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/dto/mapper"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/carts"
	"payloop/internal/lib"
	"payloop/internal/services"
)

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

	if err := c.ShouldBindJSON(&input); err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid input",
			"message": err.Error(),
		})
		return
	}

	qty := input.Quantity
	if qty <= 0 {
		qty = 1
	}

	cart, err := o.cartService.AddProduct(c.Request.Context(), carts.AddProductCommand{
		OrgId:     input.OrgId,
		CartId:    cartId,
		ProductId: input.ProductId,
		PriceId:   input.PriceId,
		Quantity:  input.Quantity,
	})
	if err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	response := mapper.ToCartResponse(cart)

	c.JSON(200, response)
}

func (o *CartController) RemoveItem(c *gin.Context) {
	var input request.RemoveItemRequest
	cartId := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid input",
			"message": err.Error(),
		})
		return
	}

	cart, err := o.cartService.RemoveItem(c.Request.Context(), carts.RemoveItemCommand{
		OrgId:  input.OrgId,
		CartId: cartId,
		Id:     input.Id,
	})
	if err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	response := mapper.ToCartResponse(cart)

	c.JSON(200, response)
}
