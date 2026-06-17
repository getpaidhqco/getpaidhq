package postgres

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type CouponRepo struct {
	db *gorm.DB
}

func NewCouponRepo(db *gorm.DB) port.CouponRepository {
	return &CouponRepo{db: db}
}

func (r *CouponRepo) Create(ctx context.Context, coupon domain.Coupon) (domain.Coupon, error) {
	row := couponRowFromDomain(coupon)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Coupon{}, err
	}
	return r.FindById(ctx, coupon.OrgId, coupon.Id)
}

func (r *CouponRepo) UpdateMutable(ctx context.Context, orgId, id, name string, active bool, metadata map[string]string) (domain.Coupon, error) {
	err := dbFromCtx(ctx, r.db).Model(&couponRow{}).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Updates(map[string]any{"name": name, "active": active, "metadata": serializeMetadata(metadata)}).Error
	if err != nil {
		return domain.Coupon{}, err
	}
	return r.FindById(ctx, orgId, id)
}

func (r *CouponRepo) FindById(ctx context.Context, orgId, id string) (domain.Coupon, error) {
	var row couponRow
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.Coupon{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CouponRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Coupon, int, error) {
	var rows []couponRow
	var total int64
	db := dbFromCtx(ctx, r.db).Model(&couponRow{}).Scopes(OrgScope(orgId))
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Scopes(Paginate(p)).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Coupon, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, int(total), nil
}

func (r *CouponRepo) DeleteIfUnreferenced(ctx context.Context, orgId, id string) error {
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&discountRow{}).
		Scopes(OrgScope(orgId)).Where("coupon_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return lib.NewCustomError(lib.BadRequestError, "coupon has discounts and cannot be deleted", nil)
	}
	return dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("id = ?", id).Delete(&couponRow{}).Error
}

// serializeMetadata marshals a metadata map for a raw column Updates call
// (GORM's map[string]any Updates path does not run the row serializer).
func serializeMetadata(m map[string]string) []byte {
	if m == nil {
		return []byte("null")
	}
	b, _ := json.Marshal(m)
	return b
}
