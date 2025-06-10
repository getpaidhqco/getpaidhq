package interfaces

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

// PaymentService defines the interface for payment operations
type PaymentService interface {
	// FindById retrieves a payment by its ID
	FindById(ctx context.Context, orgId string, id string) (entities.Payment, error)

	// List retrieves a list of payments for an organization
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Payment, int, error)

	// Refund creates a refund for a payment
	Refund(ctx context.Context, orgId string, paymentId string, input request.RefundPaymentRequest) (entities.Refund, error)
}
