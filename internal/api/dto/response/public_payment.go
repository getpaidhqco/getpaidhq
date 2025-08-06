package response

import "payloop/internal/domain/entities"

// PublicPaymentDetailsResponse represents payment link details for customers
type PublicPaymentDetailsResponse struct {
	Type     string            `json:"type"` // "invoice" or "checkout"
	Invoice  Invoice           `json:"invoice,omitempty"`
	Customer PublicCustomer    `json:"customer,omitempty"`
	Org      PublicOrgResponse `json:"org,omitempty"`
	Config   interface{}       `json:"config"`
}

// PublicCustomer represents customer data for public payment display
type PublicCustomer struct {
	Id             string            `json:"id"`
	Email          string            `json:"email"`
	FirstName      string            `json:"first_name,omitempty"`
	LastName       string            `json:"last_name,omitempty"`
	BillingAddress entities.Address  `json:"billing_address,omitempty"`
}

// PublicInvoiceResponse represents invoice data for customer display
type PublicOrgResponse struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	LogoUrl string `json:"logo_url,omitempty"`
}

// PublicOrderResponse represents order creation response for customers
type PublicOrderResponse struct {
	OrderId          string `json:"order_id"`
	PaymentProcessor string `json:"payment_processor"`
	RedirectUrl      string `json:"redirect_url,omitempty"`
	ClientSecret     string `json:"client_secret,omitempty"`
	SessionId        string `json:"session_id,omitempty"`
	Reference        string `json:"reference"`
	Amount           int    `json:"amount"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
}

// PublicOrderStatusResponse represents order status for customers
type PublicOrderStatusResponse struct {
	OrderId       string `json:"order_id"`
	Status        string `json:"status"`
	PaymentStatus string `json:"payment_status,omitempty"`
	Amount        int    `json:"amount"`
	Currency      string `json:"currency"`
}

// NewPublicCustomerFromEntity creates a new PublicCustomer response from an entity
func NewPublicCustomerFromEntity(entity entities.Customer) PublicCustomer {
	return PublicCustomer{
		Id:             entity.Id,
		Email:          entity.Email,
		FirstName:      entity.FirstName,
		LastName:       entity.LastName,
		BillingAddress: entity.BillingAddress, // Use struct directly, omitempty handles empty values
	}
}
