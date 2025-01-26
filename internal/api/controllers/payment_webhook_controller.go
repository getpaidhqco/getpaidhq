package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
	"payloop/internal/services"
	"strconv"
)

// PaymentWebhookController data type
type PaymentWebhookController struct {
	service services.PaymentWebhookService
	logger  lib.Logger
}

// NewPaymentWebhookController creates new paymentWebhook controller
func NewPaymentWebhookController(paymentWebhookService services.PaymentWebhookService, logger lib.Logger) PaymentWebhookController {
	return PaymentWebhookController{
		service: paymentWebhookService,
		logger:  logger,
	}
}

// GetPaymentWebhook gets one paymentWebhook
func (u PaymentWebhookController) GetPaymentWebhook(c *gin.Context) {
	paramID := c.Param("id")

	id, err := strconv.Atoi(paramID)
	if err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	paymentWebhook, err := u.service.GetPaymentWebhook(uint(id))

	if err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"data": paymentWebhook,
	})
}

// GetPaymentWebhooks gets all paymentWebhooks
func (u PaymentWebhookController) GetPaymentWebhooks(c *gin.Context) {
	paymentWebhooks, err := u.service.GetAllPaymentWebhooks()
	if err != nil {
		u.logger.Error(err)
	}
	c.JSON(200, gin.H{"data": paymentWebhooks})
}

// SavePaymentWebhook saves the paymentWebhook
func (u PaymentWebhookController) SavePaymentWebhook(c *gin.Context) {
	var paymentWebhook entities.PaymentWebhook

	if err := c.ShouldBindJSON(&paymentWebhook); err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := u.service.CreatePaymentWebhook(paymentWebhook); err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{"data": "paymentWebhook created"})
}

// UpdatePaymentWebhook updates paymentWebhook
func (u PaymentWebhookController) UpdatePaymentWebhook(c *gin.Context) {
	var paymentWebhook entities.PaymentWebhook

	if err := c.ShouldBindJSON(&paymentWebhook); err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := u.service.UpdatePaymentWebhook(paymentWebhook); err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{"data": "paymentWebhook updated"})
}

// DeletePaymentWebhook deletes paymentWebhook
func (u PaymentWebhookController) DeletePaymentWebhook(c *gin.Context) {
	paramID := c.Param("id")

	id, err := strconv.Atoi(paramID)
	if err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	if err := u.service.DeletePaymentWebhook(uint(id)); err != nil {
		u.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{"data": "paymentWebhook deleted"})
}
