package postgresgorm

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type PriceRepo struct {
	db *gorm.DB
}

func NewPriceRepo(db *gorm.DB) port.PriceRepository {
	return &PriceRepo{db: db}
}

func (r *PriceRepo) Create(ctx context.Context, entity domain.Price) (domain.Price, error) {
	row := priceRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Price{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PriceRepo) FindById(ctx context.Context, orgId string, id string) (domain.Price, error) {
	var row priceRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Price{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// FindByIds batch-loads prices by ID within an org. Used by services to
// hydrate read models without N+1 (e.g. OrderItemDetails composition).
func (r *PriceRepo) FindByIds(ctx context.Context, orgId string, ids []string) ([]domain.Price, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []priceRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id IN ?", ids).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return priceRowsToDomain(rows), nil
}

func (r *PriceRepo) FindByVariantId(ctx context.Context, orgId string, variantId string, p domain.Pagination) ([]domain.Price, int, error) {
	var rows []priceRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&priceRow{}).
		Scopes(OrgScope(orgId)).
		Where("variant_id = ?", variantId).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Where("variant_id = ?", variantId).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return priceRowsToDomain(rows), int(count), nil
}

// FindByVariantIds batch-loads prices across many variants. Used by Product
// read-model composition.
func (r *PriceRepo) FindByVariantIds(ctx context.Context, orgId string, variantIds []string) ([]domain.Price, error) {
	if len(variantIds) == 0 {
		return nil, nil
	}
	var rows []priceRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("variant_id IN ?", variantIds).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return priceRowsToDomain(rows), nil
}

func (r *PriceRepo) Update(ctx context.Context, entity domain.Price) (domain.Price, error) {
	row := priceRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.Price{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PriceRepo) Delete(ctx context.Context, orgId string, id string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&priceRow{}).Error
}
