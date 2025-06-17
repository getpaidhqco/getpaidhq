package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type SettingRepository interface {
	FindById(ctx context.Context, orgId string, parentId string, id string) (entities.Setting, error)
	Create(ctx context.Context, entity entities.Setting) (entities.Setting, error)
	Update(ctx context.Context, entity entities.Setting) (entities.Setting, error)
	Delete(ctx context.Context, orgId string, parentId string, id string) error
	FindAll(ctx context.Context, orgId string, parentId string) ([]entities.Setting, error)
}
