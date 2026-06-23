package postgresgorm

import (
	"context"
	"time"

	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type CouponReservationRepo struct {
	db *gorm.DB
}

func NewCouponReservationRepo(db *gorm.DB) port.CouponReservationRepository {
	return &CouponReservationRepo{db: db}
}

func (r *CouponReservationRepo) Create(ctx context.Context, res domain.CouponReservation) (domain.CouponReservation, error) {
	row := couponReservationRowFromDomain(res)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.CouponReservation{}, err
	}
	return r.findById(ctx, res.OrgId, res.Id)
}

func (r *CouponReservationRepo) FindByOrder(ctx context.Context, orgId, orderId string) ([]domain.CouponReservation, error) {
	var rows []couponReservationRow
	err := dbFromCtx(ctx, r.db).
		Where("org_id = ? AND order_id = ?", orgId, orderId).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.CouponReservation, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, nil
}

func (r *CouponReservationRepo) DeleteByOrder(ctx context.Context, orgId, orderId string) error {
	return dbFromCtx(ctx, r.db).
		Where("org_id = ? AND order_id = ?", orgId, orderId).
		Delete(&couponReservationRow{}).Error
}

func (r *CouponReservationRepo) CountLiveByCoupon(ctx context.Context, orgId, couponId string, now time.Time) (int, error) {
	var n int64
	err := dbFromCtx(ctx, r.db).Model(&couponReservationRow{}).
		Where("org_id = ? AND coupon_id = ? AND expires_at > ?", orgId, couponId, now).
		Count(&n).Error
	return int(n), err
}

func (r *CouponReservationRepo) CountLiveByCode(ctx context.Context, orgId, couponCodeId string, now time.Time) (int, error) {
	var n int64
	err := dbFromCtx(ctx, r.db).Model(&couponReservationRow{}).
		Where("org_id = ? AND coupon_code_id = ? AND expires_at > ?", orgId, couponCodeId, now).
		Count(&n).Error
	return int(n), err
}

func (r *CouponReservationRepo) ExistsLiveForCustomer(ctx context.Context, orgId, couponId, customerId string, now time.Time) (bool, error) {
	var n int64
	err := dbFromCtx(ctx, r.db).Model(&couponReservationRow{}).
		Where("org_id = ? AND coupon_id = ? AND customer_id = ? AND expires_at > ?", orgId, couponId, customerId, now).
		Limit(1).Count(&n).Error
	return n > 0, err
}

func (r *CouponReservationRepo) DeleteExpired(ctx context.Context, now time.Time) (int, error) {
	res := dbFromCtx(ctx, r.db).
		Where("expires_at <= ?", now).
		Delete(&couponReservationRow{})
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

func (r *CouponReservationRepo) findById(ctx context.Context, orgId, id string) (domain.CouponReservation, error) {
	var row couponReservationRow
	if err := dbFromCtx(ctx, r.db).Where("org_id = ? AND id = ?", orgId, id).First(&row).Error; err != nil {
		return domain.CouponReservation{}, translateErr(err)
	}
	return row.toDomain(), nil
}
