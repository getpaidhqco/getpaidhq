package controllers

import (
	"github.com/gin-gonic/gin"
	"io"
	"payloop/internal/api"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
)

// WebhookController data type
type WebhookController struct {
	webhookService services.WebhookService
	logger         logger.Logger
}

// NewWebhookController creates new user controller
func NewWebhookController(service services.WebhookService, logger logger.Logger) WebhookController {
	return WebhookController{
		webhookService: service,
		logger:         logger,
	}
}

func (u WebhookController) Process(c *gin.Context) {
	jsonData, err := io.ReadAll(c.Request.Body)

	u.logger.Debug("Processing webhook")
	err = u.webhookService.HandlePaymentWebhook(c.Request.Context(), jsonData)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, map[string]string{"status": "success"})
}
