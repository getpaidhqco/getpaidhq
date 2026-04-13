package domain

// CreateCustomerInput is the input for creating a customer.
type CreateCustomerInput struct {
	Email          string            `json:"email" binding:"required"`
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
	Psp            string            `json:"psp" binding:"required"`
	Name           string            `json:"name" binding:"required"`
	Type           PaymentMethodType `json:"type" binding:"required"`
	Details        interface{}       `json:"details"`
	Token          string            `json:"token" binding:"required"`
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
	Details         interface{}       `json:"details"`
	Token           string            `json:"token"`
	IsDefault       bool              `json:"is_default"`
	BillingAddress  Address           `json:"billing_address"`
	Metadata        map[string]string `json:"metadata"`
}
