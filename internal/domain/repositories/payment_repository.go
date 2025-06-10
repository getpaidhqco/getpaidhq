package repositories

import (
	"context"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
)

type PaymentRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Payment, error)
	FindByPspId(ctx context.Context, orgId string, id string) (entities.Payment, error)
	ListByPspId(ctx context.Context, psp common.Gateway, pspId string) ([]entities.Payment, error)
	FindBySubscriptionId(ctx context.Context, orgId string, id string, p entities.Pagination) ([]entities.Payment, int, error)
	List(ctx context.Context, orgId string, p entities.Pagination) ([]entities.Payment, int, error)
	Create(ctx context.Context, entity entities.Payment) (entities.Payment, error)
	Update(ctx context.Context, entity entities.Payment) (entities.Payment, error)

	CreateRefund(ctx context.Context, refund entities.Refund) (entities.Refund, error)
}
