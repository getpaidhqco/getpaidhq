package orders

import "payloop/internal/domain/payment_providers"

type CreateOrderCommand struct {
	OrgId    string                     `json:"org_id" binding:"required"` // TODO should be resolved from the API authn
	Customer CreateOrderCommandCustomer `json:"customer" binding:"required"`
	CartId   string                     `json:"cart_id" binding:"required"`
	Metadata map[string]string          `json:"metadata"`
}

type CompleteOrderCommand struct {
	OrgId          string                                  `json:"org_id" binding:"required"` // TODO should be resolved from the API authn
	OrderId        string                                  `json:"cart_id" binding:"required"`
	PaymentContext payment_providers.PaymentWebhookContext `json:"payment_context"`
	Metadata       map[string]string                       `json:"metadata"`
}

type CreateOrderCommandCustomer struct {
	ID       string            `json:"id"`
	Email    string            `json:"email"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}
