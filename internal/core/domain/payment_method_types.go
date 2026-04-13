package domain

import (
	"encoding/json"
	"payloop/internal/lib"
	"strconv"
	"time"
)

type PaymentMethodStatus string

const (
	PaymentMethodStatusActive  PaymentMethodStatus = "active"
	PaymentMethodStatusExpired PaymentMethodStatus = "expired"
)

type PaymentMethodType string

const (
	PaymentMethodTypeCard PaymentMethodType = "card"
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

func ParsePaymentMethodDetails(paymentMethodType PaymentMethodType, details interface{}) (PaymentMethodDetails, error) {
	switch paymentMethodType {
	case "card":
		var cardDetail CardDetail
		detailBytes, err := json.Marshal(details)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(detailBytes, &cardDetail)
		if err != nil {
			return nil, err
		}
		return cardDetail, nil

	default:
		return nil, lib.NewCustomError(lib.BadRequestError, "Invalid payment method type", nil)
	}
}
