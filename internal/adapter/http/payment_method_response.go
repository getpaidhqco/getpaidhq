package handler

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// PaymentMethodResponse is the HTTP shape of a customer's payment method.
type PaymentMethodResponse struct {
	Id             string                     `json:"id"`
	Status         domain.PaymentMethodStatus `json:"status"`
	Psp            string                     `json:"psp"`
	Name           string                     `json:"name,omitempty"`
	CustomerId     string                     `json:"customer_id"`
	BillingAddress domain.Address             `json:"billing_address"`
	Type           domain.PaymentMethodType   `json:"type"`
	Token          string                     `json:"token,omitempty"`
	Details        any                        `json:"details,omitempty"`
	Metadata       map[string]string          `json:"metadata,omitempty"`
	ExpireAt       time.Time                  `json:"expire_at,omitzero"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

// NewPaymentMethodResponse maps the aggregate to its HTTP shape.
func NewPaymentMethodResponse(pm domain.PaymentMethod) PaymentMethodResponse {
	return PaymentMethodResponse{
		Id:             pm.Id,
		Status:         pm.Status,
		Psp:            pm.Psp,
		Name:           pm.Name,
		CustomerId:     pm.CustomerId,
		BillingAddress: pm.BillingAddress,
		Type:           pm.Type,
		Token:          pm.Token,
		Details:        pm.Details,
		Metadata:       pm.Metadata,
		ExpireAt:       pm.ExpireAt,
		CreatedAt:      pm.CreatedAt,
		UpdatedAt:      pm.UpdatedAt,
	}
}
