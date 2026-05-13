package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// SubscriptionHandler handles HTTP requests for subscriptions.
type SubscriptionHandler struct {
	subsService *service.SubscriptionOrchestrationService
	logger      port.Logger
}

func NewSubscriptionHandler(subscriptionService *service.SubscriptionOrchestrationService, logger port.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{
		subsService: subscriptionService,
		logger:      logger,
	}
}

func (s *SubscriptionHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/subscriptions", option.Tags("Subscriptions"))
	fuego.Get(g, "", s.List, option.Summary("List subscriptions"))
	fuego.Get(g, "/{id}", s.Get, option.Summary("Get a subscription"))
	fuego.Get(g, "/{id}/payments", s.ListPayments, option.Summary("List subscription payments"))
	fuego.Put(g, "/{id}/pause", s.Pause, option.Summary("Pause a subscription"))
	fuego.Put(g, "/{id}/cancel", s.Cancel, option.Summary("Cancel a subscription"))
	fuego.Put(g, "/{id}/resume", s.Resume, option.Summary("Resume a subscription"))
	fuego.Patch(g, "/{id}/billing-anchor", s.UpdateBillingAnchor, option.Summary("Update subscription billing anchor"))
	fuego.Patch(g, "/{id}", s.Update, option.Summary("Update subscription metadata"))
}

func (s *SubscriptionHandler) Get(c fuego.ContextNoBody) (SubscriptionResponse, error) {
	authUser := AuthUserFrom(c)
	subscription, err := s.subsService.FindById(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return SubscriptionResponse{}, NewApiErrorFromError(err)
	}
	return NewSubscriptionFromEntity(subscription), nil
}

func (s *SubscriptionHandler) Update(c fuego.ContextWithBody[domain.UpdateSubscriptionRequest]) (domain.Subscription, error) {
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return domain.Subscription{}, err
	}

	subscription, err := s.subsService.Update(c.Context(), domain.UpdateSubscriptionInput{
		OrgId:    authUser.OrgId,
		Id:       c.PathParam("id"),
		Status:   input.Status,
		Metadata: input.Metadata,
	})
	if err != nil {
		return domain.Subscription{}, NewApiErrorFromError(err)
	}
	return subscription, nil
}

func (s *SubscriptionHandler) Pause(c fuego.ContextWithBody[PauseSubscriptionRequest]) (domain.Subscription, error) {
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return domain.Subscription{}, err
	}

	subscription, err := s.subsService.PauseSubscription(c.Context(), domain.PauseSubscriptionInput{
		OrgId:  authUser.OrgId,
		Id:     c.PathParam("id"),
		Reason: input.Reason,
	})
	if err != nil {
		return domain.Subscription{}, NewApiErrorFromError(err)
	}
	return subscription, nil
}

func (s *SubscriptionHandler) Resume(c fuego.ContextWithBody[ResumeSubscriptionRequest]) (domain.Subscription, error) {
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return domain.Subscription{}, err
	}

	subscription, err := s.subsService.ResumeSubscription(c.Context(), domain.ResumeSubscriptionInput{
		OrgId:          authUser.OrgId,
		Id:             c.PathParam("id"),
		ResumeBehavior: input.ResumeBehavior,
	})
	if err != nil {
		return domain.Subscription{}, NewApiErrorFromError(err)
	}
	return subscription, nil
}

func (s *SubscriptionHandler) Cancel(c fuego.ContextWithBody[PauseSubscriptionRequest]) (SubscriptionResponse, error) {
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return SubscriptionResponse{}, err
	}

	subscription, err := s.subsService.CancelSubscription(c.Context(), domain.CancelSubscriptionInput{
		OrgId:  authUser.OrgId,
		Id:     c.PathParam("id"),
		Reason: input.Reason,
	})
	if err != nil {
		return SubscriptionResponse{}, NewApiErrorFromError(err)
	}
	return NewSubscriptionFromEntity(subscription), nil
}

func (s *SubscriptionHandler) UpdateBillingAnchor(c fuego.ContextWithBody[UpdateBillingAnchorRequest]) (ProrationDetailsResponse, error) {
	authUser := AuthUserFrom(c)
	input, err := c.Body()
	if err != nil {
		return ProrationDetailsResponse{}, err
	}

	prorationDetails, err := s.subsService.UpdateBillingAnchor(c.Context(), domain.UpdateBillingAnchorInput{
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

	subs, total, err := s.subsService.List(c.Context(), authUser.OrgId, pagination)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	subscriptionResponses := make([]SubscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		subscriptionResponses = append(subscriptionResponses, NewSubscriptionFromEntity(sub))
	}
	return ListResponse{
		Data: subscriptionResponses,
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
