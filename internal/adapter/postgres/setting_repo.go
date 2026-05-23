package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type SettingRepo struct {
	db *gorm.DB
}

func NewSettingRepo(db *gorm.DB) port.SettingRepository {
	return &SettingRepo{db: db}
}

func (r *SettingRepo) FindById(ctx context.Context, orgId string, parentId string, id string) (domain.Setting, error) {
	var setting domain.Setting
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_id = ? AND id = ?", parentId, id).
		First(&setting).Error
	return setting, err
}

func (r *SettingRepo) Create(ctx context.Context, entity domain.Setting) (domain.Setting, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Setting{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.ParentId, entity.Id)
}
