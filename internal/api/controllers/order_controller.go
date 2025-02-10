package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	app_lib "payloop/internal/application/lib/authz"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities/orders"
	"payloop/internal/lib"
)

// OrderController data type
type OrderController struct {
	service services.OrderService
	logger  lib.Logger
	authz   app_lib.Authz
}

// NewOrderController creates new order controller
func NewOrderController(orderService services.OrderService, logger lib.Logger, authz app_lib.Authz) OrderController {
	return OrderController{
		service: orderService,
		logger:  logger,
		authz:   authz,
	}
}

func (o OrderController) CreateOrder(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	allowed := o.authz.Enforce(authUser, app_lib.CreateOrder, "")
	if !allowed {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	var input request.CreateOrderRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	order, psp, err := o.service.CreateOrderFromCart(c.Request.Context(), orders.CreateOrderInput{
		OrgId: input.OrgId,
		Customer: orders.CreateOrderCommandCustomer{
			ID:       input.Customer.ID,
			Email:    input.Customer.Email,
			Name:     input.Customer.Name,
			Metadata: nil,
		},
		CartId:   input.CartId,
		Metadata: nil,
	})
	if err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"order": order,
		"psp":   psp,
	})
}
