package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/lib"
	"strconv"
)

// OrderController data type
type OrderController struct {
	service OrderService
	logger  lib.Logger
}

// NewOrderController creates new order controller
func NewOrderController(orderService OrderService, logger lib.Logger) OrderController {
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

// SaveOrder saves the order
func (o OrderController) SaveOrder(c *gin.Context) {
	order := Order{}
	trxHandle := c.MustGet(constants.DBTransaction).(*gorm.DB)

	if err := c.ShouldBindJSON(&order); err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := o.service.WithTrx(trxHandle).CreateOrder(order); err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{"data": "order created"})
}

// UpdateOrder updates order
func (o OrderController) UpdateOrder(c *gin.Context) {
	c.JSON(200, gin.H{"data": "order updated"})
}

// DeleteOrder deletes order
func (o OrderController) DeleteOrder(c *gin.Context) {
	paramID := c.Param("id")

	id, err := strconv.Atoi(paramID)
	if err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	if err := o.service.DeleteOrder(uint(id)); err != nil {
		o.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{"data": "order deleted"})
}
