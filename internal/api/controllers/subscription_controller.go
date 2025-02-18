package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities/subscriptions"
)

// UserController data type
type SubscriptionController struct {
	subsOrchastration interfaces.SubscriptionOrchestrationService
	logger            logger.Logger
}

func NewSubscriptionController(subscriptionService interfaces.SubscriptionOrchestrationService, logger logger.Logger) SubscriptionController {
	return SubscriptionController{
		subsOrchastration: subscriptionService,
		logger:            logger,
	}
}

func (s SubscriptionController) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId
	subscriptionId := c.Param("id")

	subscription, err := s.subsOrchastration.FindById(c.Request.Context(), orgId, subscriptionId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewFromEntity(subscription))
}

// Update This only lets you change the subscription settings that have no impact on the billed amount.
func (s SubscriptionController) Update(c *gin.Context) {
	var input subscriptions.UpdateSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	subscription, err := s.subsOrchastration.Update(c.Request.Context(), subscriptions.UpdateSubscriptionInput{
		OrgId:    orgId,
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

func (s SubscriptionController) Pause(c *gin.Context) {
	var input request.PauseSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	subscription, err := s.subsOrchastration.PauseSubscription(c.Request.Context(), subscriptions.PauseSubscriptionInput{
		OrgId:  orgId,
		Id:     id,
		Reason: input.Reason,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, subscription)
}

func (s SubscriptionController) Resume(c *gin.Context) {
	var input request.ResumeSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	subscription, err := s.subsOrchastration.ResumeSubscription(c.Request.Context(), subscriptions.ResumeSubscriptionInput{
		OrgId:          orgId,
		Id:             id,
		ResumeBehavior: input.ResumeBehavior,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, subscription)
}

// Cancel a subscription
// swagger:route GET /api/subscriptions/{id}/cancel subscriptions cancelSubscription
// Cancels a subscription based on the ID
//
// Produces:
// - application/json
//
// Consumes:
// - application/json
//
// Schemes: http
//
// Responses:
// default: apiError
// 200: subscription
func (s SubscriptionController) Cancel(c *gin.Context) {
	var input request.PauseSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	subscription, err := s.subsOrchastration.CancelSubscription(c.Request.Context(), subscriptions.CancelSubscriptionInput{
		OrgId:  orgId,
		Id:     id,
		Reason: input.Reason,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, subscription)
}

// List all subscriptions
// swagger:route GET /api/subscriptions subscription listSubscriptions
// Returns a list of subscriptions based on the pagination
//
// Produces:
// - application/json
//
// Consumes:
// - application/json
//
// Schemes: http
//
// Responses:
// default: apiError
// 200: listResponse
func (s SubscriptionController) List(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	pagination := request.GetPagination(c)

	subs, total, err := s.subsOrchastration.List(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.ListResponse{
		Data: subs,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}
