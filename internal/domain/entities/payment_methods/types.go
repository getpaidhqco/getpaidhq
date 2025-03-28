package payment_methods

import (
	"strconv"
	"time"
)

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
	ExpiryMonth string `json:"expiry_month"`
	ExpiryYear  string `json:"expiry_year"`
}

func (d CardDetail) GetExpiryDate() time.Time {
	expiryYear, _ := strconv.Atoi(d.ExpiryYear)
	expiryMonth, _ := strconv.Atoi(d.ExpiryMonth)
	return time.Date(expiryYear, time.Month(expiryMonth), 1, 0, 0, 0, 0, time.UTC)
}
