package handler

import (
	"context"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
	"getpaidhq/internal/lib"
)

// SubscriptionHandler handles HTTP requests for subscriptions.
type SubscriptionHandler struct {
	subsService *service.SubscriptionOrchestrationService
	logger      port.Logger
	authz       port.Authz
}

func NewSubscriptionHandler(
	subscriptionService *service.SubscriptionOrchestrationService,
	logger port.Logger,
	authz port.Authz,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		subsService: subscriptionService,
		logger:      logger,
		authz:       authz,
	}
}

func (s *SubscriptionHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/subscriptions", option.Tags("Subscriptions"))
	fuego.Get(g, "", s.List, append(PaginationParams(), option.Summary("List subscriptions"))...)
	fuego.Get(g, "/{id}", s.Get, option.Summary("Get a subscription"))
	fuego.Get(g, "/{id}/payments", s.ListPayments, append(PaginationParams(), option.Summary("List subscription payments"))...)
	fuego.Put(g, "/{id}/pause", s.Pause, option.Summary("Pause a subscription"))
	fuego.Put(g, "/{id}/cancel", s.Cancel, option.Summary("Cancel a subscription"))
	fuego.Put(g, "/{id}/resume", s.Resume, option.Summary("Resume a subscription"))
	fuego.Patch(g, "/{id}/billing-anchor", s.UpdateBillingAnchor, option.Summary("Update subscription billing anchor"))
	fuego.Patch(g, "/{id}", s.Update, option.Summary("Update subscription metadata"))
}

// denied returns the standard 403 envelope.
func (s *SubscriptionHandler) denied() ApiError {
	return NewApiError(lib.ForbiddenError, "You are not allowed to perform this action", nil)
}

// renderDetails fetches the full SubscriptionDetails for a subscription Id.
// State-mutating handlers reuse this to assemble the nested response shape
// after a mutation returns just the aggregate.
func (s *SubscriptionHandler) renderDetails(ctx context.Context, orgId, id string) (SubscriptionResponse, error) {
	d, err := s.subsService.GetDetails(ctx, orgId, id)
	if err != nil {
		return SubscriptionResponse{}, NewApiErrorFromError(err)
	}
	return NewSubscriptionResponseFromDetails(d), nil
}

func (s *SubscriptionHandler) Get(c fuego.ContextNoBody) (SubscriptionResponse, error) {
	authUser := AuthUserFrom(c)
	return s.renderDetails(c.Context(), authUser.OrgId, c.PathParam("id"))
}

func (s *SubscriptionHandler) Update(c fuego.ContextWithBody[domain.UpdateSubscriptionRequest]) (SubscriptionResponse, error) {
	authUser := AuthUserFrom(c)
	if !s.authz.Enforce(authUser, port.ActionUpdateSubscription, c.PathParam("id")) {
		return SubscriptionResponse{}, s.denied()
	}
	input, err := c.Body()
	if err != nil {
		return SubscriptionResponse{}, err
	}
	if _, err := s.subsService.Update(c.Context(), port.UpdateSubscriptionInput{
		OrgId:    authUser.OrgId,
		Id:       c.PathParam("id"),
		Status:   input.Status,
		Metadata: input.Metadata,
	}); err != nil {
		return SubscriptionResponse{}, NewApiErrorFromError(err)
	}
	return s.renderDetails(c.Context(), authUser.OrgId, c.PathParam("id"))
}

func (s *SubscriptionHandler) Pause(c fuego.ContextWithBody[PauseSubscriptionRequest]) (SubscriptionResponse, error) {
	authUser := AuthUserFrom(c)
	if !s.authz.Enforce(authUser, port.ActionPauseSubscription, c.PathParam("id")) {
		return SubscriptionResponse{}, s.denied()
	}
	input, err := c.Body()
	if err != nil {
		return SubscriptionResponse{}, err
	}
	if _, err := s.subsService.PauseSubscription(c.Context(), port.PauseSubscriptionInput{
		OrgId:  authUser.OrgId,
		Id:     c.PathParam("id"),
		Reason: input.Reason,
	}); err != nil {
		return SubscriptionResponse{}, NewApiErrorFromError(err)
	}
	return s.renderDetails(c.Context(), authUser.OrgId, c.PathParam("id"))
}

func (s *SubscriptionHandler) Resume(c fuego.ContextWithBody[ResumeSubscriptionRequest]) (SubscriptionResponse, error) {
	authUser := AuthUserFrom(c)
	if !s.authz.Enforce(authUser, port.ActionResumeSubscription, c.PathParam("id")) {
		return SubscriptionResponse{}, s.denied()
	}
	input, err := c.Body()
	if err != nil {
		return SubscriptionResponse{}, err
	}
	if _, err := s.subsService.ResumeSubscription(c.Context(), port.ResumeSubscriptionInput{
		OrgId:          authUser.OrgId,
		Id:             c.PathParam("id"),
		ResumeBehavior: input.ResumeBehavior,
	}); err != nil {
		return SubscriptionResponse{}, NewApiErrorFromError(err)
	}
	return s.renderDetails(c.Context(), authUser.OrgId, c.PathParam("id"))
}

func (s *SubscriptionHandler) Cancel(c fuego.ContextWithBody[PauseSubscriptionRequest]) (SubscriptionResponse, error) {
	authUser := AuthUserFrom(c)
	if !s.authz.Enforce(authUser, port.ActionCancelSubscription, c.PathParam("id")) {
		return SubscriptionResponse{}, s.denied()
	}
	input, err := c.Body()
	if err != nil {
		return SubscriptionResponse{}, err
	}
	if _, err := s.subsService.CancelSubscription(c.Context(), port.CancelSubscriptionInput{
		OrgId:              authUser.OrgId,
		Id:                 c.PathParam("id"),
		Reason:             input.Reason,
		OutstandingInvoice: port.OutstandingInvoiceAction(input.OutstandingInvoice),
	}); err != nil {
		return SubscriptionResponse{}, NewApiErrorFromError(err)
	}
	return s.renderDetails(c.Context(), authUser.OrgId, c.PathParam("id"))
}

func (s *SubscriptionHandler) UpdateBillingAnchor(c fuego.ContextWithBody[UpdateBillingAnchorRequest]) (ProrationDetailsResponse, error) {
	authUser := AuthUserFrom(c)
	if !s.authz.Enforce(authUser, port.ActionUpdateBillingAnchor, c.PathParam("id")) {
		return ProrationDetailsResponse{}, s.denied()
	}
	input, err := c.Body()
	if err != nil {
		return ProrationDetailsResponse{}, err
	}

	prorationDetails, err := s.subsService.UpdateBillingAnchor(c.Context(), port.UpdateBillingAnchorInput{
		OrgId:         authUser.OrgId,
		Id:            c.PathParam("id"),
		BillingAnchor: input.BillingAnchor,
		ProrationMode: input.ProrationMode,
	})
	if err != nil {
		return ProrationDetailsResponse{}, NewApiErrorFromError(err)
	}
	return NewProrationDetailsFromEntity(prorationDetails), nil
}

func (s *SubscriptionHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	authUser := AuthUserFrom(c)
	pagination := GetPagination(c)

	details, total, err := s.subsService.ListDetails(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	rsp := make([]SubscriptionResponse, 0, len(details))
	for _, d := range details {
		rsp = append(rsp, NewSubscriptionResponseFromDetails(d))
	}
	return ListResponse{
		Data: rsp,
		Meta: Meta{Total: total, Page: pagination.Page, Limit: pagination.Limit},
	}, nil
}

func (s *SubscriptionHandler) ListPayments(c fuego.ContextNoBody) (ListResponse, error) {
	authUser := AuthUserFrom(c)
	pagination := GetPagination(c)

	payments, total, err := s.subsService.FindSubscriptionPayments(c.Context(), domain.EntityKey{
		OrgId: authUser.OrgId,
		Id:    c.PathParam("id"),
	}, pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	rsp := make([]PaymentResponse, 0, len(payments))
	for _, p := range payments {
		rsp = append(rsp, NewPaymentFromEntity(p))
	}
	return ListResponse{
		Data: rsp,
		Meta: Meta{Total: total, Page: pagination.Page, Limit: pagination.Limit},
	}, nil
}
