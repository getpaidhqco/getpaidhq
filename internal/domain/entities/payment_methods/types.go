package payment_methods

import "time"

type PaymentMethodStatus string

const (
	Active  PaymentMethodStatus = "active"
	Expired PaymentMethodStatus = "expired"
)

type PaymentMethodType string

const (
	Card PaymentMethodType = "card"
	// Add other payment method types as needed
)

type PaymentMethodDetails interface {
	GetExpiryDate() time.Time
}

type CardDetail struct {
	Brand       string `json:"brand"`
	Last4       string `json:"last4"`
	ExpiryMonth int    `json:"expiry_month"`
	ExpiryYear  int    `json:"expiry_year"`
}

func (d CardDetail) GetExpiryDate() time.Time {
	return time.Date(d.ExpiryYear, time.Month(d.ExpiryMonth), 1, 0, 0, 0, 0, time.UTC)
}
