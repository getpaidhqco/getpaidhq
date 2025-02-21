package interfaces

import "payloop/internal/api/dto/request"

type CreatePaymentMethodInput struct {
	request.CreatePaymentMethodRequest
	OrgId      string
	CustomerId string
}
