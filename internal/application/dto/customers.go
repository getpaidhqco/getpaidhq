package dto

import (
    "payloop/internal/domain/entities"
    "payloop/internal/domain/entities/payment_methods"
)

// CreateCustomerInput represents input for creating a customer
type CreateCustomerInput struct {
    Email          string                 `json:"email"`
    FirstName      string                 `json:"first_name"`
    LastName       string                 `json:"last_name"`
    BillingAddress *entities.Address      `json:"billing_address,omitempty"`
    Phone          string                 `json:"phone,omitempty"`
    Metadata       map[string]string      `json:"metadata,omitempty"`
}

// UpdateCustomerInput represents input for updating a customer
type UpdateCustomerInput struct {
    Email          *string                `json:"email,omitempty"`
    FirstName      *string                `json:"first_name,omitempty"`
    LastName       *string                `json:"last_name,omitempty"`
    BillingAddress *entities.Address      `json:"billing_address,omitempty"`
    Phone          *string                `json:"phone,omitempty"`
    Metadata       map[string]string      `json:"metadata,omitempty"`
}

// CreatePaymentMethodInput represents input for creating a payment method
type CreatePaymentMethodInput struct {
    CustomerId     string                              `json:"customer_id"`
    Psp            string                              `json:"psp"`
    Name           string                              `json:"name"`
    Type           payment_methods.PaymentMethodType   `json:"type"`
    Details        interface{}                         `json:"details"`
    Token          string                              `json:"token,omitempty"`
    IsDefault      bool                                `json:"is_default"`
    BillingAddress *entities.Address                   `json:"billing_address,omitempty"`
    Metadata       map[string]string                   `json:"metadata,omitempty"`
}

// UpdatePaymentMethodInput represents input for updating a payment method
type UpdatePaymentMethodInput struct {
    Name           *string                `json:"name,omitempty"`
    IsDefault      *bool                  `json:"is_default,omitempty"`
    BillingAddress *entities.Address      `json:"billing_address,omitempty"`
    Metadata       map[string]string      `json:"metadata,omitempty"`
}