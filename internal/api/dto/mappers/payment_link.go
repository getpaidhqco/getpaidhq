package mappers

import (
	"payloop/internal/api/dto/response"
	"payloop/internal/domain/entities"
)

// ToPaymentLinkResponse converts domain entity to API response
func ToPaymentLinkResponse(paymentLink entities.PaymentLink) response.PaymentLinkResponse {
	response := response.PaymentLinkResponse{
		Id:        paymentLink.Id,
		Slug:      paymentLink.Slug,
		Data:      paymentLink.Data,
		Config:    paymentLink.Config,
		SingleUse: paymentLink.SingleUse,
		Status:    paymentLink.Status,
		CreatedAt: paymentLink.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: paymentLink.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add optional fields if they exist
	if !paymentLink.UsedAt.IsZero() {
		usedAt := paymentLink.UsedAt.Format("2006-01-02T15:04:05Z07:00")
		response.UsedAt = usedAt
	}

	if !paymentLink.ExpiresAt.IsZero() {
		expiresAt := paymentLink.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		response.ExpiresAt = expiresAt
	}

	return response
}