package mappers

import (
	"encoding/json"
	"payloop/internal/api/dto/response"
	"payloop/internal/domain/entities"
)

// ToPaymentLinkResponse converts domain entity to API response
func ToPaymentLinkResponse(paymentLink entities.PaymentLink) response.PaymentLinkResponse {
	// Parse JSON data and config
	var data map[string]interface{}
	var config map[string]interface{}

	if len(paymentLink.Data) > 0 {
		json.Unmarshal(paymentLink.Data, &data)
	}

	if len(paymentLink.Config) > 0 {
		json.Unmarshal(paymentLink.Config, &config)
	}

	response := response.PaymentLinkResponse{
		Id:        paymentLink.Id,
		Slug:      paymentLink.Slug,
		Data:      data,
		Config:    config,
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