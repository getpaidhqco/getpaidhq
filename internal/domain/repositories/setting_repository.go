package repositories

import (
	"context"
	"payloop/internal/domain/entities"
)

type SettingRepository interface {
	FindById(ctx context.Context, orgId string, parentId string, id string) (entities.Setting, error)
	Create(ctx context.Context, entity entities.Setting) (entities.Setting, error)
}
