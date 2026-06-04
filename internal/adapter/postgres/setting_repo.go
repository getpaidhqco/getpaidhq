package postgres

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
	return setting, translateErr(err)
}

func (r *SettingRepo) Create(ctx context.Context, entity domain.Setting) (domain.Setting, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Setting{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.ParentId, entity.Id)
}

func (r *SettingRepo) Upsert(ctx context.Context, entity domain.Setting) (domain.Setting, error) {
	// Generic setting upsert (not reminder-specific). value_type is in DoUpdates
	// so a future caller writing a different Type on an existing key gets correct
	// update semantics rather than a stale type.
	err := dbFromCtx(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "org_id"}, {Name: "parent_id"}, {Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "updated_at"}),
		}).
		Create(&entity).Error
	if err != nil {
		return domain.Setting{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.ParentId, entity.Id)
}
