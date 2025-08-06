package response

import (
	"payloop/internal/domain/entities"
)

// PublicPaymentDetailsResponse represents payment link details for customers
type PublicPaymentDetailsResponse struct {
	Type          string                 `json:"type"`           // "invoice" or "checkout"
	Invoice       *PublicInvoiceResponse `json:"invoice,omitempty"`
	CheckoutItems interface{}            `json:"checkout_items,omitempty"`
	PaymentConfig interface{}            `json:"payment_config"`
	OrgId         string                 `json:"org_id"`
}

// PublicInvoiceResponse represents invoice data for customer display
type PublicInvoiceResponse struct {
	Id            string                      `json:"id"`
	DocNumber     string                      `json:"doc_number"`
	Total         int                         `json:"total"`
	AmountDue     string                      `json:"amount_due"`
	Currency      string                      `json:"currency"`
	DueAt         *string                     `json:"due_at,omitempty"`
	LineItems     []PublicInvoiceLineItem     `json:"line_items"`
	Customer      *PublicCustomerResponse     `json:"customer,omitempty"`
}

// PublicInvoiceLineItem represents line item for customer display
type PublicInvoiceLineItem struct {
	Description   string  `json:"description"`
	Quantity      string  `json:"quantity"`
	UnitPrice     int     `json:"unit_price"`
	LineTotal     int     `json:"line_total"`
}

// PublicCustomerResponse represents customer data for invoice display
type PublicCustomerResponse struct {
	Email          string             `json:"email"`
	FirstName      string             `json:"first_name,omitempty"`
	LastName       string             `json:"last_name,omitempty"`
	BillingAddress *entities.Address  `json:"billing_address,omitempty"`
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