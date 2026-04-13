package handler

import (
	"github.com/gin-gonic/gin"
)

// PaymentMethodHandler handles HTTP requests for standalone payment method endpoints.
// It delegates to the CustomerHandler for the actual implementation since payment methods
// are managed through the customer service.
type PaymentMethodHandler struct {
	customerHandler *CustomerHandler
}

// NewPaymentMethodHandler creates a new PaymentMethodHandler.
func NewPaymentMethodHandler(customerHandler *CustomerHandler) *PaymentMethodHandler {
	return &PaymentMethodHandler{
		customerHandler: customerHandler,
	}
}

// RegisterRoutes registers standalone payment method routes on the given router group.
func (s *PaymentMethodHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/payment-methods/:id", s.customerHandler.GetCustomerPaymentMethod)
}
