package mappers

import (
	"payloop/internal/api/dto/request"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

// ToCreateOrderFromInvoiceInput converts API request to application input for creating order from invoice
func ToCreateOrderFromInvoiceInput(req request.PublicCreateOrderRequest) dto.CreateOrderFromInvoiceInput {
	var billingAddress entities.Address
	if req.BillingAddress != nil {
		billingAddress = *req.BillingAddress
	}

	return dto.CreateOrderFromInvoiceInput{
		PaymentProcessor: req.PaymentProcessor,
		BillingAddress:   billingAddress,
		SuccessUrl:       req.SuccessUrl,
		CancelUrl:        req.CancelUrl,
		Metadata:         req.Metadata,
	}
}