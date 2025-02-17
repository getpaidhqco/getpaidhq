package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities/webhooks"
)

type WebhookSubscriptionController struct {
	webhookSubscriptionService services.WebhookSubscriptionService
	cartService                services.CartService
	logger                     logger.Logger
}

func NewWebhookSubscriptionController(
	webhookSubscriptionService services.WebhookSubscriptionService,
	cartService services.CartService,
	logger logger.Logger,
) WebhookSubscriptionController {

	return WebhookSubscriptionController{
		webhookSubscriptionService: webhookSubscriptionService,
		cartService:                cartService,
		logger:                     logger,
	}
}

func (s WebhookSubscriptionController) Create(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	var input request.CreateWebhookSubscriptionRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	webhook, err := s.webhookSubscriptionService.Create(c.Request.Context(), webhooks.CreateWebhookSubscriptionInput{
		OrgId:  authUser.OrgId,
		Url:    input.Url,
		Events: input.Events,
		Secret: input.Secret,
	})

	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, webhook)
}
