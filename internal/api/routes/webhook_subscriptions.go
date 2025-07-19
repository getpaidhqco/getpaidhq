package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/authn"
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/authz"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type WebhookSubscriptionRoutes struct {
	logger                        logger.Logger
	handler                       lib.RequestHandler
	webhookSubscriptionController controllers.WebhookSubscriptionController
	authz                         authz.Authz
}

// Setup user routes
func (s WebhookSubscriptionRoutes) Setup() {
	s.logger.Info("Setting up WebhookSubscription")
	api := s.handler.Gin.Group("/api")
	{
		api.POST("/webhooks", s.checkAuthz(authz.CreateWebhookSubscription), s.webhookSubscriptionController.Create)
		api.GET("/webhooks", s.checkAuthz(authz.ListWebhookSubscriptions), s.webhookSubscriptionController.List)
		api.GET("/webhooks/:id", s.checkAuthz(authz.GetWebhookSubscription), s.webhookSubscriptionController.Get)
		api.PUT("/webhooks/:id", s.checkAuthz(authz.UpdateWebhookSubscription), s.webhookSubscriptionController.Update)
		api.DELETE("/webhooks/:id", s.checkAuthz(authz.DeleteWebhookSubscription), s.webhookSubscriptionController.Delete)
	}
}

func (s WebhookSubscriptionRoutes) checkAuthz(action authz.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		authUser := user.(authn.User)
		allowed := s.authz.Enforce(authUser, action, "")
		if !allowed {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// NewWebhookSubscriptionRoutes creates new user controller
func NewWebhookSubscriptionRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	webhookSubscriptionController controllers.WebhookSubscriptionController,
	authz authz.Authz,
) WebhookSubscriptionRoutes {
	return WebhookSubscriptionRoutes{
		handler:                       handler,
		logger:                        logger,
		authz:                         authz,
		webhookSubscriptionController: webhookSubscriptionController,
	}
}
