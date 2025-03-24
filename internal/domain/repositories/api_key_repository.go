package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type ApiKeyRepository interface {
	FindById(ctx context.Context, orgId string, id string) (entities.ApiKey, error)
	FindByKey(ctx context.Context, key string) (entities.ApiKey, error)
	Create(ctx context.Context, entity entities.ApiKey) (entities.ApiKey, error)
	Update(ctx context.Context, entity entities.ApiKey) (entities.ApiKey, error)
	Delete(ctx context.Context, orgId string, id string) error
}
