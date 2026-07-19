package postgresgorm

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

func (r *ProductRepo) Find(ctx context.Context, orgId string, p domain.Pagination, statuses []domain.ProductStatus) ([]domain.Product, int, error) {
	var rows []productRow
	var count int64
	// statusScope filters by status when the caller passes one or more; an empty
	// slice means "all statuses". Applied to both the count and the page query so
	// the total matches what's returned.
	statusScope := func(db *gorm.DB) *gorm.DB {
		if len(statuses) == 0 {
			return db
		}
		return db.Where("status IN ?", statuses)
	}
	if err := dbFromCtx(ctx, r.db).Model(&productRow{}).
		Scopes(OrgScope(orgId), statusScope).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), statusScope, Paginate(p)).
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
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&productRow{}).Error
	// A product whose variant is referenced by order_items cannot be
	// hard-deleted — the FK is intentionally Restrict to preserve order
	// history. Surface that as a 409 with a clear message rather than a raw
	// SQLSTATE 23503 leaking out as an opaque 400.
	return asConflictOnFK(err, "Cannot delete a product that has existing orders.")
}
