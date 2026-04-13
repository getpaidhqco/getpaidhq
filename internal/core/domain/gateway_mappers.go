package domain

import (
	"encoding/json"
	"errors"
)

func ParsePaymentWebhookContext(data interface{}) (PaymentWebhookContext, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return PaymentWebhookContext{}, errors.New("failed to marshal data to JSON")
	}

	var payload PaymentWebhookContext
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return PaymentWebhookContext{}, errors.New("failed to unmarshal JSON to TransactionSuccessful")
	}
	return payload, nil
}
