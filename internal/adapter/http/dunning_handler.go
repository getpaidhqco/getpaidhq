package handler

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

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

func (h *DunningHandler) RegisterRoutes(s *fuego.Server) {
	campaigns := fuego.Group(s, "/dunning/campaigns", option.Tags("Dunning Campaigns"))
	fuego.Get(campaigns, "", h.ListCampaigns, option.Summary("List dunning campaigns"))
	fuego.Get(campaigns, "/{id}", h.GetCampaign, option.Summary("Get a dunning campaign"))
	fuego.Patch(campaigns, "/{id}", h.UpdateCampaign, option.Summary("Update a dunning campaign"))
	fuego.Get(campaigns, "/{id}/attempts", h.ListCampaignAttempts, option.Summary("List dunning campaign attempts"))
	fuego.Post(campaigns, "/{id}/attempts", h.TriggerManualAttempt, option.Summary("Trigger a manual dunning attempt"))
	fuego.Get(campaigns, "/{id}/communications", h.ListCampaignCommunications, option.Summary("List dunning campaign communications"))

	tokens := fuego.Group(s, "/payment-tokens", option.Tags("Payment Update Tokens"))
	fuego.Post(tokens, "/verify", h.VerifyPaymentToken, option.Summary("Verify a payment update token"))
	fuego.Post(tokens, "/activate", h.ActivatePaymentToken, option.Summary("Activate a payment update token"))

	adminTokens := fuego.Group(s, "/admin/subscriptions", option.Tags("Payment Update Tokens"))
	fuego.Post(adminTokens, "/{id}/payment-tokens", h.CreatePaymentToken, option.Summary("Admin: create a payment update token"))

	configs := fuego.Group(s, "/dunning/configurations", option.Tags("Dunning Configurations"))
	fuego.Get(configs, "", h.ListConfigurations, option.Summary("List dunning configurations"))
	fuego.Get(configs, "/{id}", h.GetConfiguration, option.Summary("Get a dunning configuration"))
	fuego.Post(configs, "", h.CreateConfiguration, option.Summary("Create a dunning configuration"))
	fuego.Patch(configs, "/{id}", h.UpdateConfiguration, option.Summary("Update a dunning configuration"))

	customers := fuego.Group(s, "/customers", option.Tags("Dunning"))
	fuego.Get(customers, "/{id}/dunning-history", h.GetCustomerDunningHistory, option.Summary("Get a customer's dunning history"))
}

type dunningList struct {
	Data  any `json:"data"`
	Total int `json:"total"`
}

// ---- Campaigns ----

func (h *DunningHandler) ListCampaigns(c fuego.ContextNoBody) (dunningList, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionListDunningCampaigns, "") {
		return dunningList{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	pagination := GetPagination(c)
	campaigns, total, err := h.dunningService.ListCampaigns(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return dunningList{}, NewApiErrorFromError(err)
	}
	out := make([]DunningCampaignResponse, 0, len(campaigns))
	for _, c := range campaigns {
		out = append(out, NewDunningCampaignResponse(c))
	}
	return dunningList{Data: out, Total: total}, nil
}

func (h *DunningHandler) GetCampaign(c fuego.ContextNoBody) (DunningCampaignResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")
	if !h.authz.Enforce(authUser, port.ActionGetDunningCampaign, id) {
		return DunningCampaignResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	campaign, err := h.dunningService.FindCampaignById(c.Context(), authUser.OrgId, id)
	if err != nil {
		return DunningCampaignResponse{}, NewApiErrorFromError(err)
	}
	return NewDunningCampaignResponse(campaign), nil
}

func (h *DunningHandler) UpdateCampaign(c fuego.ContextWithBody[UpdateDunningCampaignRequest]) (DunningCampaignResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")
	if !h.authz.Enforce(authUser, port.ActionUpdateDunningCampaign, id) {
		return DunningCampaignResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return DunningCampaignResponse{}, err
	}

	var campaign domain.DunningCampaign
	switch input.Status {
	case "paused":
		campaign, err = h.dunningService.PauseCampaign(c.Context(), domain.PauseDunningCampaignInput{
			OrgId: authUser.OrgId, CampaignId: id, Reason: input.Reason,
		})
	case "active":
		campaign, err = h.dunningService.ResumeCampaign(c.Context(), domain.ResumeDunningCampaignInput{
			OrgId: authUser.OrgId, CampaignId: id, Reason: input.Reason,
		})
	case "cancelled":
		campaign, err = h.dunningService.CancelCampaign(c.Context(), domain.CancelDunningCampaignInput{
			OrgId: authUser.OrgId, CampaignId: id, Reason: input.Reason,
		})
	default:
		return DunningCampaignResponse{}, NewApiErrorFromError(
			lib.NewCustomError(lib.BadRequestError, "Invalid status, must be one of active|paused|cancelled", nil))
	}
	if err != nil {
		return DunningCampaignResponse{}, NewApiErrorFromError(err)
	}
	return NewDunningCampaignResponse(campaign), nil
}

// ---- Attempts ----

func (h *DunningHandler) ListCampaignAttempts(c fuego.ContextNoBody) (dunningList, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")
	if !h.authz.Enforce(authUser, port.ActionGetDunningCampaign, id) {
		return dunningList{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	pagination := GetPagination(c)
	attempts, total, err := h.dunningService.ListAttemptsByCampaign(c.Context(), authUser.OrgId, id, pagination)
	if err != nil {
		return dunningList{}, NewApiErrorFromError(err)
	}
	out := make([]DunningAttemptResponse, 0, len(attempts))
	for _, a := range attempts {
		out = append(out, NewDunningAttemptResponse(a))
	}
	return dunningList{Data: out, Total: total}, nil
}

func (h *DunningHandler) TriggerManualAttempt(c fuego.ContextWithBody[TriggerManualAttemptRequest]) (DunningAttemptResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")
	if !h.authz.Enforce(authUser, port.ActionTriggerDunningAttempt, id) {
		return DunningAttemptResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return DunningAttemptResponse{}, err
	}
	attempt, err := h.dunningService.TriggerManualAttempt(c.Context(), domain.TriggerManualAttemptInput{
		OrgId:           authUser.OrgId,
		CampaignId:      id,
		PaymentMethodId: input.PaymentMethodID,
		TriggeredBy:     authUser.Id,
	})
	if err != nil {
		return DunningAttemptResponse{}, NewApiErrorFromError(err)
	}
	return NewDunningAttemptResponse(attempt), nil
}

// ---- Communications ----

func (h *DunningHandler) ListCampaignCommunications(c fuego.ContextNoBody) (dunningList, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")
	if !h.authz.Enforce(authUser, port.ActionGetDunningCampaign, id) {
		return dunningList{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	pagination := GetPagination(c)
	comms, total, err := h.dunningService.ListCommunicationsByCampaign(c.Context(), authUser.OrgId, id, pagination)
	if err != nil {
		return dunningList{}, NewApiErrorFromError(err)
	}
	out := make([]DunningCommunicationResponse, 0, len(comms))
	for _, cm := range comms {
		out = append(out, NewDunningCommunicationResponse(cm))
	}
	return dunningList{Data: out, Total: total}, nil
}

// ---- Tokens ----

func (h *DunningHandler) VerifyPaymentToken(c fuego.ContextWithBody[VerifyPaymentTokenRequest]) (PaymentUpdateTokenResponse, error) {
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return PaymentUpdateTokenResponse{}, err
	}
	token, err := h.dunningService.VerifyPaymentUpdateToken(c.Context(), authUser.OrgId, input.TokenID)
	if err != nil {
		return PaymentUpdateTokenResponse{}, NewApiErrorFromError(err)
	}
	return NewPaymentUpdateTokenResponse(token), nil
}

func (h *DunningHandler) ActivatePaymentToken(c fuego.ContextWithBody[ActivatePaymentTokenRequest]) (PaymentUpdateTokenResponse, error) {
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return PaymentUpdateTokenResponse{}, err
	}
	token, err := h.dunningService.ActivatePaymentUpdateToken(c.Context(), domain.ActivatePaymentUpdateTokenInput{
		OrgId:   authUser.OrgId,
		TokenId: input.TokenID,
		UsedIp:  clientIP(c.Request()),
	})
	if err != nil {
		return PaymentUpdateTokenResponse{}, NewApiErrorFromError(err)
	}
	return NewPaymentUpdateTokenResponse(token), nil
}

func (h *DunningHandler) CreatePaymentToken(c fuego.ContextWithBody[CreatePaymentTokenRequest]) (PaymentUpdateTokenResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")
	if !h.authz.Enforce(authUser, port.ActionCreatePaymentUpdateToken, id) {
		return PaymentUpdateTokenResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return PaymentUpdateTokenResponse{}, err
	}

	subscription, err := h.subscriptionService.FindById(c.Context(), authUser.OrgId, id)
	if err != nil {
		return PaymentUpdateTokenResponse{}, NewApiErrorFromError(err)
	}

	token, err := h.dunningService.CreatePaymentUpdateToken(c.Context(), domain.CreatePaymentUpdateTokenInput{
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
		return PaymentUpdateTokenResponse{}, NewApiErrorFromError(err)
	}
	c.SetStatus(201)
	return NewPaymentUpdateTokenResponse(token), nil
}

// ---- Configurations ----

func (h *DunningHandler) ListConfigurations(c fuego.ContextNoBody) (dunningList, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionListDunningConfigurations, "") {
		return dunningList{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	pagination := GetPagination(c)
	cfgs, total, err := h.dunningService.ListConfigurations(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return dunningList{}, NewApiErrorFromError(err)
	}
	out := make([]DunningConfigurationResponse, 0, len(cfgs))
	for _, cfg := range cfgs {
		out = append(out, NewDunningConfigurationResponse(cfg))
	}
	return dunningList{Data: out, Total: total}, nil
}

func (h *DunningHandler) GetConfiguration(c fuego.ContextNoBody) (DunningConfigurationResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")
	if !h.authz.Enforce(authUser, port.ActionGetDunningConfiguration, id) {
		return DunningConfigurationResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	cfg, err := h.dunningService.GetConfiguration(c.Context(), authUser.OrgId, id)
	if err != nil {
		return DunningConfigurationResponse{}, NewApiErrorFromError(err)
	}
	return NewDunningConfigurationResponse(cfg), nil
}

func (h *DunningHandler) CreateConfiguration(c fuego.ContextWithBody[CreateDunningConfigurationRequest]) (DunningConfigurationResponse, error) {
	authUser := AuthUserFrom(c)
	if !h.authz.Enforce(authUser, port.ActionCreateDunningConfiguration, "") {
		return DunningConfigurationResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return DunningConfigurationResponse{}, err
	}
	cfg, err := h.dunningService.CreateConfiguration(c.Context(), domain.CreateDunningConfigurationInput{
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
		return DunningConfigurationResponse{}, NewApiErrorFromError(err)
	}
	c.SetStatus(201)
	return NewDunningConfigurationResponse(cfg), nil
}

func (h *DunningHandler) UpdateConfiguration(c fuego.ContextWithBody[UpdateDunningConfigurationRequest]) (DunningConfigurationResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")
	if !h.authz.Enforce(authUser, port.ActionUpdateDunningConfiguration, id) {
		return DunningConfigurationResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	input, err := c.Body()
	if err != nil {
		return DunningConfigurationResponse{}, err
	}
	cfg, err := h.dunningService.UpdateConfiguration(c.Context(), domain.UpdateDunningConfigurationInput{
		OrgId:            authUser.OrgId,
		Id:               id,
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
		return DunningConfigurationResponse{}, NewApiErrorFromError(err)
	}
	return NewDunningConfigurationResponse(cfg), nil
}

// ---- Customer history ----

func (h *DunningHandler) GetCustomerDunningHistory(c fuego.ContextNoBody) (CustomerDunningHistoryResponse, error) {
	authUser := AuthUserFrom(c)
	id := c.PathParam("id")
	if !h.authz.Enforce(authUser, port.ActionGetCustomerDunningHistory, id) {
		return CustomerDunningHistoryResponse{}, NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
	}
	history, err := h.dunningService.GetCustomerDunningHistory(c.Context(), authUser.OrgId, id)
	if err != nil {
		return CustomerDunningHistoryResponse{}, NewApiErrorFromError(err)
	}
	return NewCustomerDunningHistoryResponse(history), nil
}

// timeOrZero is a tiny helper so DTOs don't ship as 0001-01-01T00:00:00Z.
func timeOrZero(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return t.UTC()
}

// clientIP resolves the request's originating IP using the common reverse-
// proxy headers, falling back to RemoteAddr.
func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return v
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if idx := strings.Index(v, ","); idx >= 0 {
			return strings.TrimSpace(v[:idx])
		}
		return strings.TrimSpace(v)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
