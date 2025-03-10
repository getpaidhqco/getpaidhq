package controllers

import (
	"github.com/gin-gonic/gin"
	"io"
	"payloop/internal/application/interfaces/webhooks"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
)

// WebhookController data type
type WebhookController struct {
	webhookService webhooks.WebhookService
	logger         logger.Logger
}

// NewWebhookController creates new user controller
func NewWebhookController(service webhooks.WebhookService, logger logger.Logger) WebhookController {
	return WebhookController{
		webhookService: service,
		logger:         logger,
	}
}

func (u WebhookController) Process(c *gin.Context) {
	jsonData, err := io.ReadAll(c.Request.Body)

	queryParams := c.Request.URL.Query()
	psp := queryParams.Get("p")

	u.logger.Debug("Processing webhook")
	err = u.webhookService.HandlePaymentWebhook(c.Request.Context(), webhooks.PaymentWebhookPayload{
		Psp:  common.Gateway(psp),
		Data: string(jsonData),
	})
	if err != nil {
		// we log the error and report it, but we always respond positively to the webhook
		u.logger.Errorf("Error processing webhook: %s", err.Error())
	}

	c.JSON(200, map[string]string{"status": "success"})
}
