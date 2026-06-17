package postgres

import (
	"context"

	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type CouponCodeRepo struct {
	db *gorm.DB
}

func NewCouponCodeRepo(db *gorm.DB) port.CouponCodeRepository {
	return &CouponCodeRepo{db: db}
}

func (r *CouponCodeRepo) Create(ctx context.Context, code domain.CouponCode) (domain.CouponCode, error) {
	row := couponCodeRowFromDomain(code)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.CouponCode{}, err
	}
	return r.findById(ctx, code.OrgId, code.Id)
}

func (r *CouponCodeRepo) UpdateMutable(ctx context.Context, orgId, id string, active bool, metadata map[string]string) (domain.CouponCode, error) {
	err := dbFromCtx(ctx, r.db).Model(&couponCodeRow{}).
		Scopes(OrgScope(orgId)).Where("id = ?", id).
		Updates(map[string]any{"active": active, "metadata": serializeMetadata(metadata)}).Error
	if err != nil {
		return domain.CouponCode{}, err
	}
	return r.findById(ctx, orgId, id)
}

func (r *CouponCodeRepo) IncrementRedeemed(ctx context.Context, orgId, id string) error {
	return dbFromCtx(ctx, r.db).Model(&couponCodeRow{}).
		Scopes(OrgScope(orgId)).Where("id = ?", id).
		UpdateColumn("times_redeemed", gorm.Expr("times_redeemed + 1")).Error
}

func (r *CouponCodeRepo) FindByCode(ctx context.Context, orgId, code string) (domain.CouponCode, error) {
	var row couponCodeRow
	err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).
		Where("upper(code) = upper(?)", code).First(&row).Error
	if err != nil {
		return domain.CouponCode{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CouponCodeRepo) FindByCouponId(ctx context.Context, orgId, couponId string) ([]domain.CouponCode, error) {
	var rows []couponCodeRow
	err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).
		Where("coupon_id = ?", couponId).Order("created_at desc").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.CouponCode, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, nil
}

func (r *CouponCodeRepo) findById(ctx context.Context, orgId, id string) (domain.CouponCode, error) {
	var row couponCodeRow
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.CouponCode{}, translateErr(err)
	}
	return row.toDomain(), nil
}
