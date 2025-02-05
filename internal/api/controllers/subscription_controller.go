package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities/subscriptions"
	"payloop/internal/lib"
)

// UserController data type
type SubscriptionController struct {
	subscriptionService services.SubscriptionService
	logger              lib.Logger
}

func NewSubscriptionController(subscriptionService services.SubscriptionService, logger lib.Logger) SubscriptionController {
	return SubscriptionController{
		subscriptionService: subscriptionService,
		logger:              logger,
	}
}

func (s SubscriptionController) Get(c *gin.Context) {
	// TODO
	orgId := "mollie"
	subscriptionId := c.Param("id")

	subscription, err := s.subscriptionService.FindById(c.Request.Context(), orgId, subscriptionId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, subscription)
}

func (s SubscriptionController) Update(c *gin.Context) {
	var input subscriptions.UpdateSubscriptionRequest
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	s.logger.Debug("Updating subscription", "input", input)

	subscription, err := s.subscriptionService.Update(c.Request.Context(), subscriptions.UpdateSubscriptionInput{
		OrgId:    input.OrgId,
		Id:       id,
		Status:   input.Status,
		Metadata: input.Metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, subscription)
}
