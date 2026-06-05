package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// PaymentHandler exposes read-only access to payments. Payments are created by the
// billing/PSP flow, never via the API.
type PaymentHandler struct {
	paymentService *service.PaymentService
	logger         port.Logger
	authz          port.Authz
}

func NewPaymentHandler(paymentService *service.PaymentService, logger port.Logger, authz port.Authz) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService, logger: logger, authz: authz}
}

func (h *PaymentHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/payments", option.Tags("Payments"))
	fuego.Get(g, "", h.List, option.Summary("List payments"))
	fuego.Get(g, "/{id}", h.Get, option.Summary("Get a payment"))

	subs := fuego.Group(srv, "/subscriptions", option.Tags("Payments"))
	fuego.Get(subs, "/{id}/payments", h.ListBySubscription, option.Summary("List a subscription's payments"))
}

func (h *PaymentHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, h.authz, port.ActionListPayments); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	p := GetPagination(c)
	payments, total, err := h.paymentService.List(c.Context(), authUser.OrgId, p)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	return paymentListResponse(payments, total, p), nil
}

func (h *PaymentHandler) Get(c fuego.ContextNoBody) (PaymentResponse, error) {
	if err := enforce(c, h.authz, port.ActionGetPayment); err != nil {
		return PaymentResponse{}, err
	}
	authUser := AuthUserFrom(c)
	payment, err := h.paymentService.GetById(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return PaymentResponse{}, NewApiErrorFromError(err)
	}
	return NewPaymentFromEntity(payment), nil
}

func (h *PaymentHandler) ListBySubscription(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, h.authz, port.ActionListPayments); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	p := GetPagination(c)
	payments, total, err := h.paymentService.ListBySubscription(c.Context(), authUser.OrgId, c.PathParam("id"), p)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	return paymentListResponse(payments, total, p), nil
}

func paymentListResponse(payments []domain.Payment, total int, p domain.Pagination) ListResponse {
	out := make([]PaymentResponse, len(payments))
	for i, pay := range payments {
		out[i] = NewPaymentFromEntity(pay)
	}
	return ListResponse{Data: out, Meta: Meta{Total: total, Page: p.Page, Limit: p.Limit}}
}
