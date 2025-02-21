package orders

import (
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/payment_providers"
)

type CartItem struct {
	ProductId string `json:"product_id" binding:"required"`
	PriceId   string `json:"price_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
}

type CreateOrderInput struct {
	OrgId           string                     `json:"org_id" binding:"required"`
	Customer        CreateOrderCommandCustomer `json:"customer" binding:"required"`
	SessionId       string                     `json:"session_id"`
	Currency        string                     `json:"currency"`
	CartItems       []CartItem                 `json:"items"`
	PspId           common.Gateway             `json:"psp_id" binding:"required"`
	PaymentMethodId string                     `json:"payment_method_id"`
	Metadata        map[string]string          `json:"metadata"`
	Options         map[string]string          `json:"options"`
}

type CreateOrderResponse struct {
	Order entities.Order `json:"order"`
	Psp   payment_providers.InitPaymentResponse
}

type CompleteCheckoutSessionInput struct {
	OrgId          string                                  `json:"org_id" binding:"required"` // TODO should be resolved from the API authn
	OrderId        string                                  `json:"cart_id" binding:"required"`
	PaymentContext payment_providers.PaymentWebhookContext `json:"payment_context"`
	Metadata       map[string]string                       `json:"metadata"`
}

type CreateOrderCommandCustomer struct {
	Id        string            `json:"id"`
	Email     string            `json:"email"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	Phone     string            `json:"phone"`
	Metadata  map[string]string `json:"metadata"`
}
