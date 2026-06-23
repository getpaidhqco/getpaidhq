package port

import "getpaidhq/internal/core/domain"

// CreatePaymentMethodInput is the input for PaymentMethodService.Create.
type CreatePaymentMethodInput struct {
	OrgId          string
	CustomerId     string
	Psp            string
	Name           string
	Type           domain.PaymentMethodType
	Details        any
	Token          string
	IsDefault      bool
	BillingAddress domain.Address
	Metadata       map[string]string
}

// UpdatePaymentMethodInput is the input for PaymentMethodService.Update.
type UpdatePaymentMethodInput struct {
	OrgId           string
	CustomerId      string
	PaymentMethodId string
	Name            string
	Type            domain.PaymentMethodType
	Details         any
	Token           string
	IsDefault       bool
	BillingAddress  domain.Address
	Metadata        map[string]string
}
