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
	err := r.db.WithContext(ctx).Create(&entity).Error
	if err != nil {
		return domain.Org{}, err
	}
	var created domain.Org
	err = r.db.WithContext(ctx).
		Where("id = ?", entity.Id).
		First(&created).Error
	return created, err
}
