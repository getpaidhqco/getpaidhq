package service

// CompleteCheckoutSessionInput is defined in internal/core/port/service_input.go
// because it appears in the port.OrderWorkflowService interface signature.
// See port.CompleteCheckoutSessionInput.

// CreateSessionInput is the command input for SessionService.Create.
type CreateSessionInput struct {
	OrgId    string
	Currency string
	Country  string
	Metadata map[string]string
}
