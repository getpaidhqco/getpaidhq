package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/services"
	"payloop/internal/domain/entities"
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

// Update This only lets you change the subscription settings that have no impact on the billed amount.
func (s SubscriptionController) Update(c *gin.Context) {
	var input subscriptions.UpdateSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	subscription, err := s.subscriptionService.Update(c.Request.Context(), subscriptions.UpdateSubscriptionInput{
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

	subscription, err := s.subscriptionService.Pause(c.Request.Context(), subscriptions.PauseSubscriptionInput{
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

	subscription, err := s.subscriptionService.ResumeSubscription(c.Request.Context(), subscriptions.ResumeSubscriptionInput{
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

	subscription, err := s.subscriptionService.CancelSubscription(c.Request.Context(), subscriptions.CancelSubscriptionInput{
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

// Create a new subscription in pending status
func (s SubscriptionController) Create(c *gin.Context) {
	var input request.CreateSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	s.logger.Debug("Creating subscription", "orgId", orgId, "input", input)

	subscription, err := s.subscriptionService.Create(c.Request.Context(), entities.CreateSubscriptionInput{
		OrgId:              orgId,
		PaymentMethodId:    input.PaymentMethodId,
		Amount:             input.Amount,
		Currency:           input.Currency,
		BillingInterval:    input.BillingInterval,
		BillingIntervalQty: input.BillingIntervalQty,
		Cycles:             input.Cycles,
		TrialInterval:      input.TrialInterval,
		TrialIntervalQty:   input.TrialIntervalQty,
		Metadata:           nil,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, subscription)
}

// List all subscriptions
func (s SubscriptionController) List(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	pagination := request.GetPagination(c)

	subs, err := s.subscriptionService.List(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.ListResponse{
		Data: subs,
		Meta: response.Meta{
			Total: len(subs),
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}
