package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// InvoiceHandler exposes read-only access to per-cycle invoices. Invoices are
// produced by the billing engine, never created via the API.
type InvoiceHandler struct {
	invoiceService *service.InvoiceService
	logger         port.Logger
	authz          port.Authz
}

func NewInvoiceHandler(invoiceService *service.InvoiceService, logger port.Logger, authz port.Authz) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService, logger: logger, authz: authz}
}

func (h *InvoiceHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/invoices", option.Tags("Invoices"))
	fuego.Get(g, "", h.List, append(PaginationParams(), option.Summary("List invoices"), option.OperationID("listInvoices"))...)
	fuego.Get(g, "/{id}", h.Get, option.Summary("Get an invoice"), option.OperationID("getInvoice"))

	subs := fuego.Group(srv, "/subscriptions", option.Tags("Invoices"))
	fuego.Get(subs, "/{id}/invoices", h.ListBySubscription, append(PaginationParams(), option.Summary("List a subscription's invoices"), option.OperationID("listSubscriptionInvoices"))...)
}

func (h *InvoiceHandler) List(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, h.authz, port.ActionListInvoices); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	p := GetPagination(c)
	invoices, total, err := h.invoiceService.List(c.Context(), authUser.OrgId, p)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	return invoiceListResponse(invoices, total, p), nil
}

func (h *InvoiceHandler) Get(c fuego.ContextNoBody) (InvoiceResponse, error) {
	if err := enforce(c, h.authz, port.ActionGetInvoice); err != nil {
		return InvoiceResponse{}, err
	}
	authUser := AuthUserFrom(c)
	inv, err := h.invoiceService.GetById(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return InvoiceResponse{}, NewApiErrorFromError(err)
	}
	return NewInvoiceResponse(inv), nil
}

func (h *InvoiceHandler) ListBySubscription(c fuego.ContextNoBody) (ListResponse, error) {
	if err := enforce(c, h.authz, port.ActionListInvoices); err != nil {
		return ListResponse{}, err
	}
	authUser := AuthUserFrom(c)
	p := GetPagination(c)
	invoices, total, err := h.invoiceService.ListBySubscription(c.Context(), authUser.OrgId, c.PathParam("id"), p)
	if err != nil {
		return ListResponse{}, NewApiErrorFromError(err)
	}
	return invoiceListResponse(invoices, total, p), nil
}

func invoiceListResponse(invoices []domain.Invoice, total int, p domain.Pagination) ListResponse {
	out := make([]InvoiceResponse, len(invoices))
	for i, inv := range invoices {
		out[i] = NewInvoiceResponse(inv)
	}
	return ListResponse{Data: out, Meta: Meta{Total: total, Page: p.Page, Limit: p.Limit}}
}
