package handler

import (
	"github.com/gin-gonic/gin"
)

// RegisterAllRoutes registers all handler routes under the /api group on the provided engine.
func RegisterAllRoutes(
	engine *gin.Engine,
	orderHandler *OrderHandler,
	subscriptionHandler *SubscriptionHandler,
	customerHandler *CustomerHandler,
	productHandler *ProductHandler,
	cartHandler *CartHandler,
	sessionHandler *SessionHandler,
	webhookHandler *WebhookHandler,
	webhookSubscriptionHandler *WebhookSubscriptionHandler,
	orgHandler *OrgHandler,
	healthHandler *HealthHandler,
	reportHandler *ReportHandler,
	pspHandler *PspHandler,
	userHandler *UserHandler,
	paymentMethodHandler *PaymentMethodHandler,
) {
	api := engine.Group("/api")

	orderHandler.RegisterRoutes(api)
	subscriptionHandler.RegisterRoutes(api)
	customerHandler.RegisterRoutes(api)
	productHandler.RegisterRoutes(api)
	cartHandler.RegisterRoutes(api)
	sessionHandler.RegisterRoutes(api)
	webhookHandler.RegisterRoutes(api)
	webhookSubscriptionHandler.RegisterRoutes(api)
	orgHandler.RegisterRoutes(api)
	healthHandler.RegisterRoutes(api)
	reportHandler.RegisterRoutes(api)
	pspHandler.RegisterRoutes(api)
	userHandler.RegisterRoutes(api)
	paymentMethodHandler.RegisterRoutes(api)
}
