package domain

// CreateCustomerInput is the input for creating a customer.
type CreateCustomerInput struct {
	Email          string            `json:"email" validate:"required"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	BillingAddress Address           `json:"billing_address"`
	Phone          string            `json:"phone"`
	Metadata       map[string]string `json:"metadata"`
}

// CreatePaymentMethodInput is the input for creating a payment method.
type CreatePaymentMethodInput struct {
	OrgId          string            `json:"org_id"`
	CustomerId     string            `json:"customer_id"`
	Psp            string            `json:"psp" validate:"required"`
	Name           string            `json:"name" validate:"required"`
	Type           PaymentMethodType `json:"type" validate:"required"`
	Details        any               `json:"details"`
	Token          string            `json:"token" validate:"required"`
	IsDefault      bool              `json:"is_default"`
	BillingAddress Address           `json:"billing_address"`
	Metadata       map[string]string `json:"metadata"`
}

// UpdatePaymentMethodInput is the input for updating a payment method.
type UpdatePaymentMethodInput struct {
	OrgId           string            `json:"org_id"`
	CustomerId      string            `json:"customer_id"`
	PaymentMethodId string            `json:"payment_method_id"`
	Name            string            `json:"name"`
	Type            PaymentMethodType `json:"type"`
	Details         any               `json:"details"`
	Token           string            `json:"token"`
	IsDefault       bool              `json:"is_default"`
	BillingAddress  Address           `json:"billing_address"`
	Metadata        map[string]string `json:"metadata"`
}
