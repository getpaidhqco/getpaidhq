package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/domain/orders"
	"payloop/internal/lib"
	"payloop/internal/services"
	"strconv"
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

// GetOneOrder gets one order
func (o OrderController) GetOneOrder(c *gin.Context) {
	paramID := c.Param("id")

	id, err := strconv.Atoi(paramID)
	if err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	order, err := o.service.GetOneOrder(uint(id))

	if err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"data": order,
	})
}

// GetOrders gets all orders
func (o OrderController) GetOrders(c *gin.Context) {
	orders, err := o.service.GetAllOrders()
	if err != nil {
		o.logger.Error(err)
	}
	c.JSON(200, gin.H{"data": orders})
}

func (o OrderController) CreateOrder(c *gin.Context) {
	var input orders.CreateOrderInput

	if err := c.ShouldBindJSON(&input); err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	order, err := o.service.CreateOrder(c.Request.Context(), input)
	if err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, order)
}
