package request

import (
	"payloop/internal/domain/entities"
)

type CreateCustomerRequest struct {
	Email          string            `json:"email" binding:"required"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	BillingAddress entities.Address  `json:"billing_address"`
	Phone          string            `json:"phone"`
	Metadata       map[string]string `json:"metadata"`
}

type CreatePaymentMethodRequest struct {
	Psp            string            `json:"psp" binding:"required"`
	Name           string            `json:"name" binding:"required"`
	Type           string            `json:"type" binding:"required"`
	Token          string            `json:"token" binding:"required"`
	IsDefault      bool              `json:"is_default"`
	BillingAddress entities.Address  `json:"billing_address"`
	Metadata       map[string]string `json:"details"`
}
