package repositories

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
)

type OrgRepository interface {
	Create(ctx context.Context, entity entities.Org) (entities.Org, error)
	FindById(ctx context.Context, id string) (entities.Org, error)
	Update(ctx context.Context, entity entities.Org) (entities.Org, error)
	List(ctx context.Context, pagination dto.Pagination) ([]entities.Org, int, error)
}
