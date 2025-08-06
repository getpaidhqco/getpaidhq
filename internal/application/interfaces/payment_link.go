package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payment_links"
)

type PaymentLinkService interface {
	// Payment Link CRUD operations
	GetPaymentLink(ctx context.Context, orgId string, id string) (entities.PaymentLink, error)
	GetPaymentLinkBySlug(ctx context.Context, slug string) (entities.PaymentLink, error)
	ListPaymentLinks(ctx context.Context, orgId string, pagination dto.Pagination) (dto.PaginatedResult[entities.PaymentLink], error)
	CreatePaymentLink(ctx context.Context, orgId string, input payment_links.CreatePaymentLinkInput) (payment_links.PaymentLinkCreationResult, error)
	UpdatePaymentLink(ctx context.Context, orgId string, id string, input payment_links.UpdatePaymentLinkInput) (entities.PaymentLink, error)
	DeletePaymentLink(ctx context.Context, orgId string, id string) error

	// Public Access operations
	ValidatePaymentLinkAccess(ctx context.Context, slug, token string) (entities.PaymentLink, error)

	// Payment Link Usage operations
	RecordPaymentLinkUsage(ctx context.Context, orgId string, input payment_links.RecordPaymentLinkUsageInput) (entities.PaymentLinkUsage, error)
	GetPaymentLinkUsage(ctx context.Context, orgId string, id string) (entities.PaymentLinkUsage, error)
	ListPaymentLinkUsages(ctx context.Context, orgId string, paymentLinkId string) ([]entities.PaymentLinkUsage, error)
}
