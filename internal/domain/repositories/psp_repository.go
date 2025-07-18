package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type PspRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.Gateway, error)
	Create(ctx context.Context, input entities.Gateway) (entities.Gateway, error)
	Update(ctx context.Context, input entities.Gateway) (entities.Gateway, error)
	Delete(ctx context.Context, orgId string, id string) error
	FindAll(ctx context.Context, orgId string) ([]entities.Gateway, error)
}
