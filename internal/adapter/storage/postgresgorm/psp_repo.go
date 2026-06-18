package postgresgorm

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type PspRepo struct {
	db *gorm.DB
}

func NewPspRepo(db *gorm.DB) port.PspRepository {
	return &PspRepo{db: db}
}

func (r *PspRepo) FindById(ctx context.Context, orgId string, id string) (domain.PspConfig, error) {
	var row pspConfigRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.PspConfig{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *PspRepo) Create(ctx context.Context, input domain.PspConfig) (domain.PspConfig, error) {
	row := pspConfigRowFromDomain(input)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.PspConfig{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
