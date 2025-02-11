package controllers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities/webhooks"
	"payloop/internal/lib"
)

type WebhookSubscriptionController struct {
	webhookSubscriptionService services.WebhookSubscriptionService
	cartService                services.CartService
	logger                     lib.Logger
}

func NewWebhookSubscriptionController(webhookSubscriptionService services.WebhookSubscriptionService, cartService services.CartService, logger lib.Logger) WebhookSubscriptionController {
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	webhook, err := s.webhookSubscriptionService.Create(c.Request.Context(), webhooks.CreateWebhookSubscriptionInput{
		OrgId:  authUser.OrgId,
		Url:    input.Url,
		Events: input.Events,
		Secret: input.Secret,
	})

	if err != nil {
		var serr lib.CustomError
		if errors.As(err, &serr) {
			if serr.Type == lib.NotFoundError {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   serr.Type,
					"message": serr.Message,
					"details": serr.Err,
				})
				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "An internal error occurred.",
			"details": err.Error(),
		})
		return
	}

	c.JSON(200, webhook)
}
