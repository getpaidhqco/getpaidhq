package service

import "getpaidhq/internal/core/domain"

// CreateCustomerInput is the command input for CustomerService.Create.
type CreateCustomerInput struct {
	Email          string
	FirstName      string
	LastName       string
	BillingAddress domain.Address
	Phone          string
	Metadata       map[string]string
}
