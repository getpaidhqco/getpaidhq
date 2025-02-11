package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/authn"
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/authz"
	"payloop/internal/lib"
)

type WebhookSubscriptionRoutes struct {
	logger                        lib.Logger
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
		api.GET("/webhooks", s.checkAuthz(authz.ListWebhookSubscriptions), s.webhookSubscriptionController.Create)
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
	logger lib.Logger,
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
