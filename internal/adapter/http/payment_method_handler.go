package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
)

// PaymentMethodHandler exposes the standalone /payment-methods/{id} GET
// endpoint by delegating to the CustomerHandler implementation.
type PaymentMethodHandler struct {
	customerHandler *CustomerHandler
}

func NewPaymentMethodHandler(customerHandler *CustomerHandler) *PaymentMethodHandler {
	return &PaymentMethodHandler{customerHandler: customerHandler}
}

func (s *PaymentMethodHandler) RegisterRoutes(srv *fuego.Server) {
	g := fuego.Group(srv, "/payment-methods", option.Tags("Payment Methods"))
	fuego.Get(g, "/{id}", s.customerHandler.GetCustomerPaymentMethod, option.Summary("Get a payment method"))
}
