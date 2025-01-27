package controllers

import (
	"payloop/internal/application/services"
	"payloop/internal/lib"
)

// PaymentWebhookController data type
type PaymentWebhookController struct {
	service services.WebhookService
	logger  lib.Logger
}

// NewPaymentWebhookController creates new paymentWebhook controller
func NewPaymentWebhookController(paymentWebhookService services.WebhookService, logger lib.Logger) PaymentWebhookController {
	return PaymentWebhookController{
		service: paymentWebhookService,
		logger:  logger,
	}
}
