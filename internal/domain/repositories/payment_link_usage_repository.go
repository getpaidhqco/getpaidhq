package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type PaymentLinkUsageRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.PaymentLinkUsage, error)
	ListByPaymentLinkId(ctx context.Context, orgId string, paymentLinkId string) ([]entities.PaymentLinkUsage, error)
	Create(ctx context.Context, input entities.PaymentLinkUsage) (entities.PaymentLinkUsage, error)
}