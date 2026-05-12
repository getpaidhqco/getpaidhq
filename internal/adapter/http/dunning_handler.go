package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

type DunningHandler struct {
	dunningService      *service.DunningOrchestrationService
	subscriptionService *service.SubscriptionService
	logger              port.Logger
	authz               port.Authz
}

func NewDunningHandler(
	dunningService *service.DunningOrchestrationService,
	subscriptionService *service.SubscriptionService,
	logger port.Logger,
	authz port.Authz,
) *DunningHandler {
	return &DunningHandler{
		dunningService:      dunningService,
		subscriptionService: subscriptionService,
		logger:              logger,
		authz:               authz,
	}
}

func (h *DunningHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/dunning/campaigns", h.ListCampaigns)
	rg.GET("/dunning/campaigns/:id", h.GetCampaign)
	rg.PATCH("/dunning/campaigns/:id", h.UpdateCampaign)
	rg.GET("/dunning/campaigns/:id/attempts", h.ListCampaignAttempts)
	rg.POST("/dunning/campaigns/:id/attempts", h.TriggerManualAttempt)
	rg.GET("/dunning/campaigns/:id/communications", h.ListCampaignCommunications)

	rg.POST("/payment-tokens/verify", h.VerifyPaymentToken)
	rg.POST("/payment-tokens/activate", h.ActivatePaymentToken)
	rg.POST("/admin/subscriptions/:id/payment-tokens", h.CreatePaymentToken)

	rg.GET("/dunning/configurations", h.ListConfigurations)
	rg.GET("/dunning/configurations/:id", h.GetConfiguration)
	rg.POST("/dunning/configurations", h.CreateConfiguration)
	rg.PATCH("/dunning/configurations/:id", h.UpdateConfiguration)

	rg.GET("/customers/:id/dunning-history", h.GetCustomerDunningHistory)
}

// ---- Campaigns ----

func (h *DunningHandler) ListCampaigns(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionListDunningCampaigns, "") {
		writeNotAllowed(c)
		return
	}
	pagination := GetPagination(c)
	campaigns, total, err := h.dunningService.ListCampaigns(c.Request.Context(), authUser.OrgId, pagination)
	if err != nil {
		writeApiErr(c, err)
		return
	}
	out := make([]DunningCampaignResponse, 0, len(campaigns))
	for _, c := range campaigns {
		out = append(out, NewDunningCampaignResponse(c))
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "total": total})
}

func (h *DunningHandler) GetCampaign(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionGetDunningCampaign, c.Param("id")) {
		writeNotAllowed(c)
		return
	}
	campaign, err := h.dunningService.FindCampaignById(c.Request.Context(), authUser.OrgId, c.Param("id"))
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusOK, NewDunningCampaignResponse(campaign))
}

func (h *DunningHandler) UpdateCampaign(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionUpdateDunningCampaign, c.Param("id")) {
		writeNotAllowed(c)
		return
	}
	var input UpdateDunningCampaignRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		writeApiErr(c, err)
		return
	}

	var campaign domain.DunningCampaign
	var err error
	switch input.Status {
	case "paused":
		campaign, err = h.dunningService.PauseCampaign(c.Request.Context(), domain.PauseDunningCampaignInput{
			OrgId:      authUser.OrgId,
			CampaignId: c.Param("id"),
			Reason:     input.Reason,
		})
	case "active":
		campaign, err = h.dunningService.ResumeCampaign(c.Request.Context(), domain.ResumeDunningCampaignInput{
			OrgId:      authUser.OrgId,
			CampaignId: c.Param("id"),
			Reason:     input.Reason,
		})
	case "cancelled":
		campaign, err = h.dunningService.CancelCampaign(c.Request.Context(), domain.CancelDunningCampaignInput{
			OrgId:      authUser.OrgId,
			CampaignId: c.Param("id"),
			Reason:     input.Reason,
		})
	default:
		writeApiErr(c, lib.NewCustomError(lib.BadRequestError, "Invalid status, must be one of active|paused|cancelled", nil))
		return
	}
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusOK, NewDunningCampaignResponse(campaign))
}

// ---- Attempts ----

func (h *DunningHandler) ListCampaignAttempts(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionGetDunningCampaign, c.Param("id")) {
		writeNotAllowed(c)
		return
	}
	pagination := GetPagination(c)
	attempts, total, err := h.dunningService.ListAttemptsByCampaign(c.Request.Context(), authUser.OrgId, c.Param("id"), pagination)
	if err != nil {
		writeApiErr(c, err)
		return
	}
	out := make([]DunningAttemptResponse, 0, len(attempts))
	for _, a := range attempts {
		out = append(out, NewDunningAttemptResponse(a))
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "total": total})
}

func (h *DunningHandler) TriggerManualAttempt(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionTriggerDunningAttempt, c.Param("id")) {
		writeNotAllowed(c)
		return
	}
	var input TriggerManualAttemptRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		writeApiErr(c, err)
		return
	}
	attempt, err := h.dunningService.TriggerManualAttempt(c.Request.Context(), domain.TriggerManualAttemptInput{
		OrgId:           authUser.OrgId,
		CampaignId:      c.Param("id"),
		PaymentMethodId: input.PaymentMethodID,
		TriggeredBy:     authUser.Id,
	})
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusOK, NewDunningAttemptResponse(attempt))
}

// ---- Communications ----

func (h *DunningHandler) ListCampaignCommunications(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionGetDunningCampaign, c.Param("id")) {
		writeNotAllowed(c)
		return
	}
	pagination := GetPagination(c)
	comms, total, err := h.dunningService.ListCommunicationsByCampaign(c.Request.Context(), authUser.OrgId, c.Param("id"), pagination)
	if err != nil {
		writeApiErr(c, err)
		return
	}
	out := make([]DunningCommunicationResponse, 0, len(comms))
	for _, cm := range comms {
		out = append(out, NewDunningCommunicationResponse(cm))
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "total": total})
}

// ---- Tokens ----

func (h *DunningHandler) VerifyPaymentToken(c *gin.Context) {
	authUser := mustAuthUser(c)
	var input VerifyPaymentTokenRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		writeApiErr(c, err)
		return
	}
	token, err := h.dunningService.VerifyPaymentUpdateToken(c.Request.Context(), authUser.OrgId, input.TokenID)
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusOK, NewPaymentUpdateTokenResponse(token))
}

func (h *DunningHandler) ActivatePaymentToken(c *gin.Context) {
	authUser := mustAuthUser(c)
	var input ActivatePaymentTokenRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		writeApiErr(c, err)
		return
	}
	token, err := h.dunningService.ActivatePaymentUpdateToken(c.Request.Context(), domain.ActivatePaymentUpdateTokenInput{
		OrgId:   authUser.OrgId,
		TokenId: input.TokenID,
		UsedIp:  c.ClientIP(),
	})
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusOK, NewPaymentUpdateTokenResponse(token))
}

func (h *DunningHandler) CreatePaymentToken(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionCreatePaymentUpdateToken, c.Param("id")) {
		writeNotAllowed(c)
		return
	}
	var input CreatePaymentTokenRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		writeApiErr(c, err)
		return
	}

	// Resolve the subscription to find the customer id.
	subscription, err := h.subscriptionService.FindById(c.Request.Context(), authUser.OrgId, c.Param("id"))
	if err != nil {
		writeApiErr(c, err)
		return
	}

	token, err := h.dunningService.CreatePaymentUpdateToken(c.Request.Context(), domain.CreatePaymentUpdateTokenInput{
		OrgId:          authUser.OrgId,
		SubscriptionId: subscription.Id,
		CustomerId:     subscription.CustomerId,
		MaxUses:        input.MaxUses,
		ExpiryHours:    input.ExpiryHours,
		AllowedActions: input.AllowedActions,
		AdminGenerated: true,
		AdminUserId:    authUser.Id,
		AdminReason:    input.AdminReason,
		AdminNotes:     input.AdminNotes,
		CreatedBy:      authUser.Id,
	})
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, NewPaymentUpdateTokenResponse(token))
}

// ---- Configurations ----

func (h *DunningHandler) ListConfigurations(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionListDunningConfigurations, "") {
		writeNotAllowed(c)
		return
	}
	pagination := GetPagination(c)
	cfgs, total, err := h.dunningService.ListConfigurations(c.Request.Context(), authUser.OrgId, pagination)
	if err != nil {
		writeApiErr(c, err)
		return
	}
	out := make([]DunningConfigurationResponse, 0, len(cfgs))
	for _, cfg := range cfgs {
		out = append(out, NewDunningConfigurationResponse(cfg))
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "total": total})
}

func (h *DunningHandler) GetConfiguration(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionGetDunningConfiguration, c.Param("id")) {
		writeNotAllowed(c)
		return
	}
	cfg, err := h.dunningService.GetConfiguration(c.Request.Context(), authUser.OrgId, c.Param("id"))
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusOK, NewDunningConfigurationResponse(cfg))
}

func (h *DunningHandler) CreateConfiguration(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionCreateDunningConfiguration, "") {
		writeNotAllowed(c)
		return
	}
	var input CreateDunningConfigurationRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		writeApiErr(c, err)
		return
	}
	cfg, err := h.dunningService.CreateConfiguration(c.Request.Context(), domain.CreateDunningConfigurationInput{
		OrgId:            authUser.OrgId,
		Name:             input.Name,
		Description:      input.Description,
		Priority:         input.Priority,
		AppliesTo:        input.AppliesTo,
		TargetRules:      input.TargetRules,
		Config:           input.Config,
		IsAbTest:         input.IsAbTest,
		AbTestPercentage: input.AbTestPercentage,
		CreatedBy:        authUser.Id,
	})
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, NewDunningConfigurationResponse(cfg))
}

func (h *DunningHandler) UpdateConfiguration(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionUpdateDunningConfiguration, c.Param("id")) {
		writeNotAllowed(c)
		return
	}
	var input UpdateDunningConfigurationRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		writeApiErr(c, err)
		return
	}
	cfg, err := h.dunningService.UpdateConfiguration(c.Request.Context(), domain.UpdateDunningConfigurationInput{
		OrgId:            authUser.OrgId,
		Id:               c.Param("id"),
		Name:             input.Name,
		Description:      input.Description,
		Priority:         input.Priority,
		AppliesTo:        input.AppliesTo,
		TargetRules:      input.TargetRules,
		Config:           input.Config,
		Status:           input.Status,
		IsAbTest:         input.IsAbTest,
		AbTestPercentage: input.AbTestPercentage,
	})
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusOK, NewDunningConfigurationResponse(cfg))
}

// ---- Customer history ----

func (h *DunningHandler) GetCustomerDunningHistory(c *gin.Context) {
	authUser := mustAuthUser(c)
	if !h.authz.Enforce(authUser, port.ActionGetCustomerDunningHistory, c.Param("id")) {
		writeNotAllowed(c)
		return
	}
	history, err := h.dunningService.GetCustomerDunningHistory(c.Request.Context(), authUser.OrgId, c.Param("id"))
	if err != nil {
		writeApiErr(c, err)
		return
	}
	c.JSON(http.StatusOK, NewCustomerDunningHistoryResponse(history))
}

// ---- helpers ----

func mustAuthUser(c *gin.Context) port.AuthUser {
	user, _ := c.Get("user")
	return user.(port.AuthUser)
}

func writeApiErr(c *gin.Context, err error) {
	apiErr := NewApiErrorFromError(err)
	c.JSON(apiErr.GetHttpErrorCode(), apiErr)
}

func writeNotAllowed(c *gin.Context) {
	apiErr := NewApiError(lib.AuthenticationError, "You are not allowed to perform this action", nil)
	c.JSON(apiErr.GetHttpErrorCode(), apiErr)
}

// timeOrZero is a tiny helper so DTOs don't ship as 0001-01-01T00:00:00Z.
func timeOrZero(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return t.UTC()
}
