package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type WebhookRoutes struct {
	logger            logger.Logger
	handler           lib.RequestHandler
	webhookController controllers.WebhookController
}

// Setup user routes
func (s WebhookRoutes) Setup() {
	s.logger.Info("Setting up Webhook")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/notify", s.webhookController.Process)
	}
}

// NewWebhookRoutes creates new user controller
func NewWebhookRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	webhookController controllers.WebhookController,
) WebhookRoutes {
	return WebhookRoutes{
		handler:           handler,
		logger:            logger,
		webhookController: webhookController,
	}
}
