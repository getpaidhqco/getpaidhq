package request

import (
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payment_methods"
)

type CreateCustomerRequest struct {
	Email          string            `json:"email" binding:"required,email"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	BillingAddress entities.Address  `json:"billing_address"`
	Phone          string            `json:"phone" binding:"omitempty,e164"`
	Metadata       map[string]string `json:"metadata"`
}

type CreatePaymentMethodRequest struct {
	Psp  string `json:"psp" binding:"required"`
	Name string `json:"name" binding:"required"`

	// Type of payment method, e.g. card, bank account, etc.
	Type           payment_methods.PaymentMethodType `json:"type" binding:"required"`
	Details        interface{}                       `json:"details"`
	Token          string                            `json:"token" binding:"required"`
	IsDefault      bool                              `json:"is_default"`
	BillingAddress entities.Address                  `json:"billing_address"`
	Metadata       map[string]string                 `json:"metadata"`
}
type UpdatePaymentMethodRequest struct {
	Name           string                            `json:"name"`
	Type           payment_methods.PaymentMethodType `json:"type"`
	Details        interface{}                       `json:"details"`
	Token          string                            `json:"token"`
	IsDefault      bool                              `json:"is_default"`
	BillingAddress entities.Address                  `json:"billing_address"`
	Metadata       map[string]string                 `json:"metadata"`
}
