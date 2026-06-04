package postgres

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type ProductRepo struct {
	db *gorm.DB
}

func NewProductRepo(db *gorm.DB) port.ProductRepository {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) FindById(ctx context.Context, orgId string, id string) (domain.Product, error) {
	var row productRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Product{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *ProductRepo) Create(ctx context.Context, product domain.Product) (domain.Product, error) {
	row := productRowFromDomain(product)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Product{}, err
	}
	return r.FindById(ctx, product.OrgId, product.Id)
}

func (r *ProductRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Product, int, error) {
	var rows []productRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&productRow{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Product, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, int(count), nil
}

func (r *ProductRepo) Update(ctx context.Context, product domain.Product) (domain.Product, error) {
	row := productRowFromDomain(product)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.Product{}, err
	}
	return r.FindById(ctx, product.OrgId, product.Id)
}

func (r *ProductRepo) Delete(ctx context.Context, orgId string, id string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&productRow{}).Error
}
