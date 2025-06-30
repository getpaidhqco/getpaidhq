package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"payloop/internal/api"
	"payloop/internal/api/authn"
	"payloop/internal/api/dto/request"
	"payloop/internal/api/dto/response"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities/dunning"
)

// DunningController handles HTTP requests related to dunning
type DunningController struct {
	dunningService      interfaces.DunningOrchestrationService
	subscriptionService interfaces.SubscriptionService
	logger              logger.Logger
}

// NewDunningController creates a new DunningController
func NewDunningController(
	dunningService interfaces.DunningOrchestrationService,
	subscriptionService interfaces.SubscriptionService,
	logger logger.Logger,
) DunningController {
	return DunningController{
		dunningService:      dunningService,
		subscriptionService: subscriptionService,
		logger:              logger,
	}
}

// ListCampaigns returns a list of dunning campaigns
func (d DunningController) ListCampaigns(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	pagination := request.GetPagination(c)

	campaigns, total, err := d.dunningService.ListCampaigns(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	campaignResponses := make([]response.DunningCampaignResponse, len(campaigns))
	for i, campaign := range campaigns {
		campaignResponses[i] = response.FromDunningCampaign(campaign)
	}

	c.JSON(http.StatusOK, response.ListResponse{
		Data: campaignResponses,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// GetCampaign returns a dunning campaign by ID
func (d DunningController) GetCampaign(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	campaign, err := d.dunningService.FindCampaignById(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, response.FromDunningCampaign(campaign))
}

// UpdateCampaign updates a dunning campaign
func (d DunningController) UpdateCampaign(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	var req request.UpdateDunningCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	var campaign dunning.DunningCampaign
	var err error

	switch req.Status {
	case string(dunning.DunningStatusPaused):
		campaign, err = d.dunningService.PauseCampaign(c.Request.Context(), interfaces.PauseDunningCampaignInput{
			OrgId:  orgId,
			Id:     id,
			Reason: req.Reason,
		})
	case string(dunning.DunningStatusActive):
		campaign, err = d.dunningService.ResumeCampaign(c.Request.Context(), interfaces.ResumeDunningCampaignInput{
			OrgId:  orgId,
			Id:     id,
			Reason: req.Reason,
		})
	case string(dunning.DunningStatusCancelled):
		campaign, err = d.dunningService.CancelCampaign(c.Request.Context(), interfaces.CancelDunningCampaignInput{
			OrgId:  orgId,
			Id:     id,
			Reason: req.Reason,
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, response.FromDunningCampaign(campaign))
}

// ListCampaignAttempts returns a list of attempts for a dunning campaign
func (d DunningController) ListCampaignAttempts(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	campaignId := c.Param("id")
	pagination := request.GetPagination(c)

	attempts, total, err := d.dunningService.ListAttemptsByCampaign(c.Request.Context(), orgId, campaignId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	attemptResponses := make([]response.DunningAttemptResponse, len(attempts))
	for i, attempt := range attempts {
		attemptResponses[i] = response.FromDunningAttempt(attempt)
	}

	c.JSON(http.StatusOK, response.ListResponse{
		Data: attemptResponses,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// TriggerManualAttempt triggers a manual payment attempt for a dunning campaign
func (d DunningController) TriggerManualAttempt(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	campaignId := c.Param("id")

	var req request.TriggerManualAttemptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	attempt, err := d.dunningService.TriggerManualAttempt(c.Request.Context(), interfaces.TriggerManualAttemptInput{
		OrgId:           orgId,
		CampaignId:      campaignId,
		PaymentMethodId: req.PaymentMethodID,
		TriggeredBy:     user.(authn.User).Id,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, response.FromDunningAttempt(attempt))
}

// ListCampaignCommunications returns a list of communications for a dunning campaign
func (d DunningController) ListCampaignCommunications(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	campaignId := c.Param("id")
	pagination := request.GetPagination(c)

	communications, total, err := d.dunningService.ListCommunicationsByCampaign(c.Request.Context(), orgId, campaignId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	communicationResponses := make([]response.DunningCommunicationResponse, len(communications))
	for i, communication := range communications {
		communicationResponses[i] = response.FromDunningCommunication(communication)
	}

	c.JSON(http.StatusOK, response.ListResponse{
		Data: communicationResponses,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// VerifyPaymentToken verifies a payment token without consuming usage
func (d DunningController) VerifyPaymentToken(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId

	var req request.VerifyPaymentTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	token, err := d.dunningService.VerifyPaymentUpdateToken(c.Request.Context(), orgId, req.TokenID)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, response.FromPaymentUpdateToken(token))
}

// ActivatePaymentToken activates a payment token and creates a session
func (d DunningController) ActivatePaymentToken(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId

	var req request.ActivatePaymentTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	token, err := d.dunningService.ActivatePaymentUpdateToken(c.Request.Context(), interfaces.ActivatePaymentUpdateTokenInput{
		OrgId:     orgId,
		TokenId:   req.TokenID,
		IpAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, response.FromPaymentUpdateToken(token))
}

// CreatePaymentToken creates a payment token for a subscription
func (d DunningController) CreatePaymentToken(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	subscriptionId := c.Param("id")

	var req request.CreatePaymentTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	// Get customer ID from subscription
	subscription, err := d.subscriptionService.FindById(c.Request.Context(), orgId, subscriptionId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	token, err := d.dunningService.CreatePaymentUpdateToken(c.Request.Context(), interfaces.CreatePaymentUpdateTokenInput{
		OrgId:          orgId,
		SubscriptionId: subscriptionId,
		CustomerId:     subscription.CustomerId,
		MaxUses:        req.MaxUses,
		ExpiryHours:    req.ExpiryHours,
		AllowedActions: req.AllowedActions,
		AdminGenerated: true,
		AdminUserId:    user.(authn.User).Id,
		AdminReason:    req.AdminReason,
		AdminNotes:     req.AdminNotes,
		CreatedBy:      user.(authn.User).Id,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusCreated, response.FromPaymentUpdateToken(token))
}

// ListConfigurations returns a list of dunning configurations
func (d DunningController) ListConfigurations(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	pagination := request.GetPagination(c)

	configs, total, err := d.dunningService.ListConfigurations(c.Request.Context(), orgId, pagination)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	configResponses := make([]response.DunningConfigurationResponse, len(configs))
	for i, config := range configs {
		configResponses[i] = response.FromDunningConfiguration(config)
	}

	c.JSON(http.StatusOK, response.ListResponse{
		Data: configResponses,
		Meta: response.Meta{
			Total: total,
			Page:  pagination.Page,
			Limit: pagination.Limit,
		},
	})
}

// GetConfiguration returns a dunning configuration by ID
func (d DunningController) GetConfiguration(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	config, err := d.dunningService.GetConfiguration(c.Request.Context(), orgId, id)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, response.FromDunningConfiguration(config))
}

// CreateConfiguration creates a new dunning configuration
func (d DunningController) CreateConfiguration(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId

	var req request.CreateDunningConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	config, err := d.dunningService.CreateConfiguration(c.Request.Context(), interfaces.CreateDunningConfigurationInput{
		OrgId:            orgId,
		Name:             req.Name,
		Description:      req.Description,
		Priority:         req.Priority,
		AppliesTo:        req.AppliesTo,
		TargetRules:      req.TargetRules,
		Config:           req.Config,
		IsAbTest:         req.IsAbTest,
		AbTestPercentage: req.AbTestPercentage,
		CreatedBy:        user.(authn.User).Id,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusCreated, response.FromDunningConfiguration(config))
}

// UpdateConfiguration updates a dunning configuration
func (d DunningController) UpdateConfiguration(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	id := c.Param("id")

	var req request.UpdateDunningConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	config, err := d.dunningService.UpdateConfiguration(c.Request.Context(), interfaces.UpdateDunningConfigurationInput{
		OrgId:            orgId,
		Id:               id,
		Name:             req.Name,
		Description:      req.Description,
		Priority:         req.Priority,
		AppliesTo:        req.AppliesTo,
		TargetRules:      req.TargetRules,
		Config:           req.Config,
		Status:           req.Status,
		IsAbTest:         req.IsAbTest,
		AbTestPercentage: req.AbTestPercentage,
	})
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, response.FromDunningConfiguration(config))
}

// GetCustomerDunningHistory returns the dunning history for a customer
func (d DunningController) GetCustomerDunningHistory(c *gin.Context) {
	user, _ := c.Get("user")
	orgId := user.(authn.User).OrgId
	customerId := c.Param("id")

	history, err := d.dunningService.GetCustomerDunningHistory(c.Request.Context(), orgId, customerId)
	if err != nil {
		apiErr := api.NewApiErrorFromError(err)
		c.JSON(apiErr.GetHttpErrorCode(), apiErr)
		return
	}

	c.JSON(http.StatusOK, response.FromCustomerDunningHistory(history))
}
