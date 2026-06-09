package port

import "getpaidhq/internal/core/domain"

// CreateCustomerInput is the command input for CustomerService.Create.
//
// JSON tags are load-bearing: fuego derives the request body's OpenAPI schema from
// them, and the request decoder matches incoming keys against them. Without tags the
// generated CreateCustomerInput schema is empty and the decoder only accepts the Go
// field names — so snake_case bodies (first_name, billing_address, …) were rejected
// with 400 "unknown field", breaking customer creation from the dashboard/SDK.
type CreateCustomerInput struct {
	Email          string            `json:"email" validate:"required"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	BillingAddress domain.Address    `json:"billing_address"`
	Phone          string            `json:"phone"`
	Metadata       map[string]string `json:"metadata"`
}
