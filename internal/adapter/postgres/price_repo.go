package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type PriceRepo struct {
	db *gorm.DB
}

func NewPriceRepo(db *gorm.DB) port.PriceRepository {
	return &PriceRepo{db: db}
}

func (r *PriceRepo) Create(ctx context.Context, entity domain.Price) (domain.Price, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Price{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PriceRepo) FindById(ctx context.Context, orgId string, id string) (domain.Price, error) {
	var price domain.Price
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&price).Error
	return price, translateErr(err)
}

func (r *PriceRepo) FindByVariantId(ctx context.Context, orgId string, variantId string, p domain.Pagination) ([]domain.Price, int, error) {
	var prices []domain.Price
	var count int64
	err := dbFromCtx(ctx, r.db).Model(&domain.Price{}).
		Scopes(OrgScope(orgId)).
		Where("variant_id = ?", variantId).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Where("variant_id = ?", variantId).
		Find(&prices).Error
	return prices, int(count), err
}

func (r *PriceRepo) Update(ctx context.Context, entity domain.Price) (domain.Price, error) {
	err := dbFromCtx(ctx, r.db).Save(&entity).Error
	if err != nil {
		return domain.Price{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PriceRepo) Delete(ctx context.Context, orgId string, id string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&domain.Price{}).Error
}
