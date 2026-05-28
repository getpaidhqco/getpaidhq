package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type PspRepo struct {
	db *gorm.DB
}

func NewPspRepo(db *gorm.DB) port.PspRepository {
	return &PspRepo{db: db}
}

func (r *PspRepo) FindById(ctx context.Context, orgId string, id string) (domain.PspConfig, error) {
	var config domain.PspConfig
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&config).Error
	return config, translateErr(err)
}

func (r *PspRepo) Create(ctx context.Context, input domain.PspConfig) (domain.PspConfig, error) {
	err := dbFromCtx(ctx, r.db).Create(&input).Error
	if err != nil {
		return domain.PspConfig{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
