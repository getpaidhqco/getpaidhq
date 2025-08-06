package request

import "payloop/internal/domain/entities"

// PublicCreateOrderRequest represents a public order creation request from a payment link
type PublicCreateOrderRequest struct {
	PaymentProcessor string                `json:"payment_processor" binding:"required"`
	CustomerEmail    string                `json:"customer_email,omitempty"`
	CustomerName     string                `json:"customer_name,omitempty"`
	BillingAddress   *entities.Address     `json:"billing_address,omitempty"`
	SuccessUrl       string                `json:"success_url,omitempty"`
	CancelUrl        string                `json:"cancel_url,omitempty"`
	Metadata         map[string]string     `json:"metadata,omitempty"`
}