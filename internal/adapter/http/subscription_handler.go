package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/core/service"
)

// SubscriptionHandler handles HTTP requests for subscriptions.
type SubscriptionHandler struct {
	subsService *service.SubscriptionService
	logger      port.Logger
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(subscriptionService *service.SubscriptionService, logger port.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{
		subsService: subscriptionService,
		logger:      logger,
	}
}

// RegisterRoutes registers subscription routes on the given router group.
func (s *SubscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/subscriptions", s.List)
	rg.GET("/subscriptions/:id", s.Get)
	rg.GET("/subscriptions/:id/payments", s.ListPayments)
	rg.PUT("/subscriptions/:id/pause", s.Pause)
	rg.PUT("/subscriptions/:id/cancel", s.Cancel)
	rg.PUT("/subscriptions/:id/resume", s.Resume)
	rg.PATCH("/subscriptions/:id/billing-anchor", s.UpdateBillingAnchor)
	rg.PATCH("/subscriptions/:id", s.Update)
}

func (s *SubscriptionHandler) Get(c *gin.Context) {
	user, _ := c.Get("user")
	authUser := user.(port.AuthUser)
	orgId := authUser.OrgId
	subscriptionId := c.Param("id")

	subscription, err := s.subsService.FindById(c.Request.Context(), orgId, subscriptionId)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, NewSubscriptionFromEntity(subscription))
}

// Update lets you change subscription settings that have no impact on the billed amount.
func (s *SubscriptionHandler) Update(c *gin.Context) {
	var input domain.UpdateSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	subscription, err := s.subsService.Update(c.Request.Context(), domain.UpdateSubscriptionInput{
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

func (s *SubscriptionHandler) Pause(c *gin.Context) {
	var input PauseSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	subscription, err := s.subsService.PauseSubscription(c.Request.Context(), domain.PauseSubscriptionInput{
		OrgId:  orgId,
		Id:     id,
		Reason: input.Reason,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, subscription)
}

func (s *SubscriptionHandler) Resume(c *gin.Context) {
	var input ResumeSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	subscription, err := s.subsService.ResumeSubscription(c.Request.Context(), domain.ResumeSubscriptionInput{
		OrgId:          orgId,
		Id:             id,
		ResumeBehavior: input.ResumeBehavior,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, subscription)
}

func (s *SubscriptionHandler) Cancel(c *gin.Context) {
	var input PauseSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	subscription, err := s.subsService.CancelSubscription(c.Request.Context(), domain.CancelSubscriptionInput{
		OrgId:  orgId,
		Id:     id,
		Reason: input.Reason,
	})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, NewSubscriptionFromEntity(subscription))
}

func (s *SubscriptionHandler) UpdateBillingAnchor(c *gin.Context) {
	var input UpdateBillingAnchorRequest
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	prorationDetails, err := s.subsService.UpdateBillingAnchor(
		c.Request.Context(),
		domain.UpdateBillingAnchorInput{
			OrgId:         orgId,
			Id:            id,
			BillingAnchor: input.BillingAnchor,
			ProrationMode: input.ProrationMode,
		})
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, NewProrationDetailsFromEntity(prorationDetails))
}

func (s *SubscriptionHandler) List(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	pagination := GetPagination(c)

	subs, total, err := s.subsService.List(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	var subscriptionResponses = make([]SubscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		subscriptionResponses = append(subscriptionResponses, NewSubscriptionFromEntity(sub))
	}

	c.JSON(200, ListResponse{
		Data: subscriptionResponses,
		Meta: Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

func (s *SubscriptionHandler) ListPayments(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(port.AuthUser).OrgId
	pagination := GetPagination(c)
	id := c.Param("id")

	payments, total, err := s.subsService.FindSubscriptionPayments(c.Request.Context(), domain.EntityKey{
		OrgId: orgId,
		Id:    id,
	}, pagination)
	if err != nil {
		apiErr := NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	var rsp []PaymentResponse
	for _, p := range payments {
		rsp = append(rsp, NewPaymentFromEntity(p))
	}

	c.JSON(200, ListResponse{
		Data: rsp,
		Meta: Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}
