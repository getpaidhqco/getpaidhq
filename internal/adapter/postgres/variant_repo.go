package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type VariantRepo struct {
	db *gorm.DB
}

func NewVariantRepo(db *gorm.DB) port.VariantRepository {
	return &VariantRepo{db: db}
}

func (r *VariantRepo) Create(ctx context.Context, entity domain.Variant) (domain.Variant, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Variant{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *VariantRepo) FindById(ctx context.Context, orgId string, id string) (domain.Variant, error) {
	var variant domain.Variant
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Preload("Prices").
		First(&variant).Error
	return variant, err
}

func (r *VariantRepo) FindByProductId(ctx context.Context, orgId string, productId string, p domain.Pagination) ([]domain.Variant, int, error) {
	var variants []domain.Variant
	var count int64
	err := dbFromCtx(ctx, r.db).Model(&domain.Variant{}).
		Scopes(OrgScope(orgId)).
		Where("product_id = ?", productId).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Where("product_id = ?", productId).
		Preload("Prices").
		Find(&variants).Error
	return variants, int(count), err
}

func (r *VariantRepo) Update(ctx context.Context, entity domain.Variant) (domain.Variant, error) {
	err := dbFromCtx(ctx, r.db).Save(&entity).Error
	if err != nil {
		return domain.Variant{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *VariantRepo) Delete(ctx context.Context, orgId string, id string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&domain.Variant{}).Error
}
