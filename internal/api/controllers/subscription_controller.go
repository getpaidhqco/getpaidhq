package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/mappers"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/dto"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
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

func (s SubscriptionController) Create(c *gin.Context) {
	var input request.CreateSubscriptionRequest
	user, _ := c.Get("user")
	authUser := user.(authn.User)
	orgId := authUser.OrgId

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Convert API request to application DTO
	appInput := mappers.ToCreateSubscriptionInput(input)

	// Call the service to create the subscription
	subscription, err := s.subsOrchastration.Create(c.Request.Context(), orgId, appInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusCreated, response.NewSubscriptionFromEntity(subscription))
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

	c.JSON(200, response.NewSubscriptionFromEntity(subscription))
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
		OrgId:                orgId,
		Id:                   id,
		Status:               input.Status,
		DefaultPaymentMethod: input.DefaultPaymentMethod,
		Metadata:             input.Metadata,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewSubscriptionFromEntity(subscription))
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

	c.JSON(200, response.NewSubscriptionFromEntity(subscription))
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

	c.JSON(200, response.NewSubscriptionFromEntity(subscription))
}

// Cancel a subscription
// swagger:route GET /api/subscriptions/{id}/cancel subscriptions cancelSubscription
// Cancels a subscription based on the Id
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

	c.JSON(200, response.NewSubscriptionFromEntity(subscription))
}

func (s SubscriptionController) UpdateBillingAnchor(c *gin.Context) {
	var input request.UpdateBillingAnchorRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	result, err := s.subsOrchastration.UpdateBillingAnchor(
		c.Request.Context(),
		dto.UpdateBillingAnchorInput{
			OrgId:         orgId,
			Id:            id,
			BillingAnchor: input.BillingAnchor,
			ProrationMode: dto.ProrationMode(input.ProrationMode),
		})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewUpdateBillingAnchorResultFromDto(result))
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
	var subscriptionResponses = make([]response.Subscription, 0, len(subs))
	for _, sub := range subs {
		subscriptionResponses = append(subscriptionResponses, response.NewSubscriptionFromEntity(sub))
	}

	c.JSON(200, response.ListResponse{
		Data: subscriptionResponses,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

func (s SubscriptionController) ListPayments(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	pagination := request.GetPagination(c)
	id := c.Param("id")

	payments, total, err := s.subsOrchastration.FindSubscriptionPayments(c.Request.Context(), entities.EntityKey{
		OrgId: orgId,
		Id:    id,
	}, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}
	var rsp []response.Payment
	for _, p := range payments {
		rsp = append(rsp, response.NewPaymentFromEntity(p))
	}

	c.JSON(200, response.ListResponse{
		Data: rsp,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// ChangePlan changes a subscription's plan to a different variant/price
// swagger:route PUT /api/subscriptions/{id}/change-plan subscriptions changePlan
// Changes a subscription's plan to a different variant/price
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
func (s SubscriptionController) ChangePlan(c *gin.Context) {
	var input request.ChangePlanRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	subscription, _, err := s.subsOrchastration.ChangeSubscriptionPlan(c.Request.Context(), subscriptions.ChangePlanInput{
		OrgId:         orgId,
		Id:            id,
		NewVariantId:  input.NewVariantId,
		NewPriceId:    input.NewPriceId,
		ProrationMode: input.ProrationMode,
		EffectiveDate: input.EffectiveDate,
		Reason:        input.Reason,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewSubscriptionFromEntity(*subscription))
}

// Activate a subscription
// swagger:route PUT /api/subscriptions/{id}/activate subscriptions activateSubscription
// Activates a subscription based on the Id
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
func (s SubscriptionController) Activate(c *gin.Context) {
	var input request.ActivateSubscriptionRequest
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	if err := c.ShouldBindJSON(&input); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	activateInput := subscriptions.ActivateSubscriptionInput{
		OrgId:          orgId,
		Id:             id,
		PaymentMethodId: input.PaymentMethodId,
		Reason:         input.Reason,
	}

	subscription, err := s.subsOrchastration.Activate(c.Request.Context(), activateInput)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(200, response.NewSubscriptionFromEntity(subscription))
}
