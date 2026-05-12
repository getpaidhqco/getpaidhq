package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// WebhookSubscriptionHandler handles HTTP requests for webhook subscriptions.
type WebhookSubscriptionHandler struct {
	webhookSubscriptionService *service.WebhookSubscriptionService
	logger                     port.Logger
	authz                      port.Authz
}

// NewWebhookSubscriptionHandler creates a new WebhookSubscriptionHandler.
func NewWebhookSubscriptionHandler(
	webhookSubscriptionService *service.WebhookSubscriptionService,
	logger port.Logger,
	authz port.Authz,
) *WebhookSubscriptionHandler {
	return &WebhookSubscriptionHandler{
		webhookSubscriptionService: webhookSubscriptionService,
		logger:                     logger,
		authz:                      authz,
	}
}

// RegisterRoutes registers webhook subscription routes on the given router group.
func (s *WebhookSubscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/webhooks", s.checkAuthz(port.ActionCreateWebhookSubscription), s.Create)
	rg.GET("/webhooks", s.checkAuthz(port.ActionListWebhookSubscriptions), s.Create)
}

func (s *WebhookSubscriptionHandler) checkAuthz(action port.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		authUser := user.(port.AuthUser)
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

func (s *WebhookSubscriptionHandler) Create(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	var input CreateWebhookSubscriptionRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	webhook, err := s.webhookSubscriptionService.Create(c.Request.Context(), domain.CreateWebhookSubscriptionInput{
		OrgId:  authUser.OrgId,
		Url:    input.Url,
		Events: input.Events,
		Secret: input.Secret,
	})

	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, webhook)
}
