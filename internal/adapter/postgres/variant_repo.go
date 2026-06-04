package postgres

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type VariantRepo struct {
	db *gorm.DB
}

func NewVariantRepo(db *gorm.DB) port.VariantRepository {
	return &VariantRepo{db: db}
}

func (r *VariantRepo) Create(ctx context.Context, entity domain.Variant) (domain.Variant, error) {
	row := variantRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Variant{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *VariantRepo) FindById(ctx context.Context, orgId string, id string) (domain.Variant, error) {
	var row variantRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Variant{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *VariantRepo) FindByProductId(ctx context.Context, orgId string, productId string, p domain.Pagination) ([]domain.Variant, int, error) {
	var rows []variantRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&variantRow{}).
		Scopes(OrgScope(orgId)).
		Where("product_id = ?", productId).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Where("product_id = ?", productId).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return variantRowsToDomain(rows), int(count), nil
}

func (r *VariantRepo) Update(ctx context.Context, entity domain.Variant) (domain.Variant, error) {
	row := variantRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.Variant{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *VariantRepo) Delete(ctx context.Context, orgId string, id string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&variantRow{}).Error
}
