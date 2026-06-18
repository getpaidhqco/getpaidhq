package postgresgorm

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
	var row settingRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_id = ? AND id = ?", parentId, id).
		First(&row).Error
	if err != nil {
		return domain.Setting{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *SettingRepo) Create(ctx context.Context, entity domain.Setting) (domain.Setting, error) {
	row := settingRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Setting{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.ParentId, entity.Id)
}

func (r *SettingRepo) List(ctx context.Context, orgId string, parentId string, p domain.Pagination) ([]domain.Setting, int, error) {
	var rows []settingRow
	var count int64
	countQ := dbFromCtx(ctx, r.db).Model(&settingRow{}).Scopes(OrgScope(orgId))
	listQ := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId), Paginate(p))
	if parentId != "" {
		countQ = countQ.Where("parent_id = ?", parentId)
		listQ = listQ.Where("parent_id = ?", parentId)
	}
	if err := countQ.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := listQ.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return settingRowsToDomain(rows), int(count), nil
}

func (r *SettingRepo) Delete(ctx context.Context, orgId string, parentId string, id string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_id = ? AND id = ?", parentId, id).
		Delete(&settingRow{}).Error
}

func (r *SettingRepo) Upsert(ctx context.Context, entity domain.Setting) (domain.Setting, error) {
	// Generic setting upsert (not reminder-specific). value_type is in DoUpdates
	// so a future caller writing a different Type on an existing key gets correct
	// update semantics rather than a stale type.
	row := settingRowFromDomain(entity)
	err := dbFromCtx(ctx, r.db).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "org_id"}, {Name: "parent_id"}, {Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "updated_at"}),
		}).
		Create(&row).Error
	if err != nil {
		return domain.Setting{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.ParentId, entity.Id)
}
