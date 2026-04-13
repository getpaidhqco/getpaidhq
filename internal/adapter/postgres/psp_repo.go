package postgres

import (
	"context"

	"gorm.io/gorm"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

type PspRepo struct {
	db *gorm.DB
}

func NewPspRepo(db *gorm.DB) port.PspRepository {
	return &PspRepo{db: db}
}

func (r *PspRepo) FindById(ctx context.Context, orgId string, id string) (domain.PspConfig, error) {
	var config domain.PspConfig
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&config).Error
	return config, err
}

func (r *PspRepo) Create(ctx context.Context, input domain.PspConfig) (domain.PspConfig, error) {
	err := r.db.WithContext(ctx).Create(&input).Error
	if err != nil {
		return domain.PspConfig{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
