package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/core/service"
	"payloop/internal/lib"
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
		authUser, err := getAuthUser(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
			c.Abort()
			return
		}
		allowed := s.authz.Enforce(authUser, action, "")
		if !allowed {
			apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
			c.JSON(apiErr.GetHttpErrorCode(), apiErr)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *WebhookSubscriptionHandler) Create(c *gin.Context) {
	authUser, err := getAuthUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, NewApiError("authentication_error", err.Error(), nil))
		return
	}
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
