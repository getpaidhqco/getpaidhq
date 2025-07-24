package repositories

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/domain/entities"
)

type PaymentLinkRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.PaymentLink, error)
	FindBySlug(ctx context.Context, slug string) (entities.PaymentLink, error)
	List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.PaymentLink, int, error)
	Create(ctx context.Context, input entities.PaymentLink) (entities.PaymentLink, error)
	Update(ctx context.Context, input entities.PaymentLink) (entities.PaymentLink, error)
	Delete(ctx context.Context, orgId string, id string) error
}
