package payment_methods

import (
	"encoding/json"
	"payloop/internal/lib"
)

func ParseDetails(paymentMethodType PaymentMethodType, details interface{}) (PaymentMethodDetails, error) {
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
