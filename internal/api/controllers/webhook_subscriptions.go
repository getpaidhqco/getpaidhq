package controllers

import (
	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities/webhooks"
)

type WebhookSubscriptionController struct {
	webhookSubscriptionService interfaces.WebhookSubscriptionService
	cartService                services.CartService
	logger                     logger.Logger
}

func NewWebhookSubscriptionController(
	webhookSubscriptionService interfaces.WebhookSubscriptionService,
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

func (s WebhookSubscriptionController) List(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	pagination := request.GetPagination(c)

	webhooks, total, err := s.webhookSubscriptionService.List(c.Request.Context(), authUser.OrgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.ListResponse{
		Data: webhooks,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

func (s WebhookSubscriptionController) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	id := c.Param("id")

	webhook, err := s.webhookSubscriptionService.GetByID(c.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, webhook)
}

func (s WebhookSubscriptionController) Update(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	id := c.Param("id")
	var input request.UpdateWebhookSubscriptionRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// First get the existing webhook to ensure it exists and belongs to the org
	existingWebhook, err := s.webhookSubscriptionService.GetByID(c.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Update the webhook with the new values
	existingWebhook.URL = input.Url
	existingWebhook.Events = input.Events
	existingWebhook.Secret = input.Secret

	updatedWebhook, err := s.webhookSubscriptionService.Update(c.Request.Context(), existingWebhook)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, updatedWebhook)
}

func (s WebhookSubscriptionController) Delete(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	id := c.Param("id")

	err := s.webhookSubscriptionService.Delete(c.Request.Context(), authUser.OrgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(204, nil)
}
