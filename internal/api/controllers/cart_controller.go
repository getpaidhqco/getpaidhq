package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/mapper"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities/carts"
)

var validate *validator.Validate

// CartController data type
type CartController struct {
	cartService     services.CartService
	discountService interfaces.DiscountService
	logger          logger.Logger
}

// NewCartController creates new Cart controller
func NewCartController(cartService services.CartService, discountService interfaces.DiscountService, logger logger.Logger) CartController {
	return CartController{
		cartService:     cartService,
		discountService: discountService,
		logger:          logger,
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

// ValidateCoupon validates a coupon code and applies it to the cart if valid
func (o *CartController) ValidateCoupon(c *gin.Context) {
	var input request.ValidateCouponRequest
	cartId := c.Param("id")
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// First, get the cart to calculate the total amount
	cartEntity, err := o.cartService.GetCart(orgId, cartId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Validate the coupon code
	validationResult, err := o.discountService.ValidateDiscountCode(c.Request.Context(), orgId, dto.ValidateDiscountCodeInput{
		Code:     input.Code,
		Amount:   int(cartEntity.Total),
		Currency: "USD", // Assuming USD as default currency
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	if !validationResult.Valid {
		c.JSON(400, gin.H{
			"error":   "Invalid coupon code",
			"message": validationResult.Message,
		})
		return
	}

	// Apply the discount to the cart
	_, err = o.discountService.ApplyDiscount(c.Request.Context(), orgId, dto.ApplyDiscountInput{
		DiscountId:   validationResult.DiscountId,
		ResourceType: "cart",
		ResourceId:   cartId,
		Amount:       int(cartEntity.Total),
		Currency:     "USD", // Assuming USD as default currency
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Get the updated cart
	updatedCart, err := o.cartService.GetCart(orgId, cartId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	response := mapper.ToCartResponse(updatedCart)
	c.JSON(200, response)
}
