package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/orders"
	"payloop/internal/lib"
	"payloop/internal/services"
)

// OrderController data type
type OrderController struct {
	service services.OrderService
	logger  lib.Logger
}

// NewOrderController creates new order controller
func NewOrderController(orderService services.OrderService, logger lib.Logger) OrderController {
	return OrderController{
		service: orderService,
		logger:  logger,
	}
}

func (o OrderController) CreateOrder(c *gin.Context) {
	var input request.CreateOrderRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	order, err := o.service.CreateOrder(c.Request.Context(), orders.CreateOrderCommand{
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

	c.JSON(200, order)
}
