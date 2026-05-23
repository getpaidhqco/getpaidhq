package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type ProductRepo struct {
	db *gorm.DB
}

func NewProductRepo(db *gorm.DB) port.ProductRepository {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) FindById(ctx context.Context, orgId string, id string) (domain.Product, error) {
	var product domain.Product
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Preload("Variants").
		Preload("Variants.Prices").
		First(&product).Error
	return product, err
}

func (r *ProductRepo) Create(ctx context.Context, product domain.Product) (domain.Product, error) {
	err := dbFromCtx(ctx, r.db).Create(&product).Error
	if err != nil {
		return domain.Product{}, err
	}
	return r.FindById(ctx, product.OrgId, product.Id)
}

func (r *ProductRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Product, int, error) {
	var products []domain.Product
	var count int64
	err := dbFromCtx(ctx, r.db).Model(&domain.Product{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Preload("Variants").
		Preload("Variants.Prices").
		Find(&products).Error
	return products, int(count), err
}

func (r *ProductRepo) Update(ctx context.Context, product domain.Product) (domain.Product, error) {
	err := dbFromCtx(ctx, r.db).Save(&product).Error
	if err != nil {
		return domain.Product{}, err
	}
	return r.FindById(ctx, product.OrgId, product.Id)
}

func (r *ProductRepo) Delete(ctx context.Context, orgId string, id string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&domain.Product{}).Error
}
