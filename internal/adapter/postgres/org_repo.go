package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type OrgRepo struct {
	db *gorm.DB
}

func NewOrgRepo(db *gorm.DB) port.OrgRepository {
	return &OrgRepo{db: db}
}

func (r *OrgRepo) Create(ctx context.Context, entity domain.Org) (domain.Org, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Org{}, err
	}
	var created domain.Org
	err = dbFromCtx(ctx, r.db).
		Where("id = ?", entity.Id).
		First(&created).Error
	return created, translateErr(err)
}

func (r *OrgRepo) ListIds(ctx context.Context) ([]string, error) {
	var ids []string
	err := dbFromCtx(ctx, r.db).
		Model(&domain.Org{}).
		Where("status = ?", domain.OrgStatusActive).
		Pluck("id", &ids).Error
	return ids, err
}
