package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	"getpaidhq/internal/core/service"
)

// PaymentMethodHandler exposes the flat /payment-methods/{id} GET endpoint.
// Payment methods are looked up by org + id (the read is authn-gated and
// org-scoped, not role-gated), so this handler depends only on the
// CustomerService — not on any sibling handler.
type PaymentMethodHandler struct {
	customerService *service.CustomerService
}

func NewPaymentMethodHandler(customerService *service.CustomerService) *PaymentMethodHandler {
	return &PaymentMethodHandler{customerService: customerService}
}

func (s *PaymentMethodHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/payment-methods", option.Tags("Payment Methods"))
	fuego.Get(g, "/{id}", s.Get, option.Summary("Get a payment method"), option.OperationID("getPaymentMethod"))
}

func (s *PaymentMethodHandler) Get(c fuego.ContextNoBody) (PaymentMethodResponse, error) {
	authUser := AuthUserFrom(c)
	pm, err := s.customerService.GetPaymentMethod(c.Context(), authUser.OrgId, c.PathParam("id"))
	if err != nil {
		return PaymentMethodResponse{}, NewApiErrorFromError(err)
	}
	return NewPaymentMethodResponse(pm), nil
}
