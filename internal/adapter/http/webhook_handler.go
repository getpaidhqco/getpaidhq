package handler

import (
	"io"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/core/service"
)

// WebhookHandler handles incoming payment webhook requests.
type WebhookHandler struct {
	webhookService *service.WebhookService
	logger         port.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(service *service.WebhookService, logger port.Logger) *WebhookHandler {
	return &WebhookHandler{
		webhookService: service,
		logger:         logger,
	}
}

// RegisterRoutes registers webhook routes on the given router group.
func (u *WebhookHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/notify", u.Process)
	rg.POST("/notify/cdc", u.Process)
}

func (u *WebhookHandler) Process(c *gin.Context) {
	jsonData, err := io.ReadAll(c.Request.Body)
	_ = err

	queryParams := c.Request.URL.Query()
	psp := queryParams.Get("p")

	u.logger.Debug("processing webhook")
	err = u.webhookService.HandlePaymentWebhook(c.Request.Context(), port.PaymentWebhookPayload{
		Psp:  domain.Gateway(psp),
		Data: string(jsonData),
	})
	if err != nil {
		// log the error and report it, but always respond positively to the webhook
		u.logger.Error("error processing webhook", "error", err)
	}

	c.JSON(200, map[string]string{"status": "success"})
}
