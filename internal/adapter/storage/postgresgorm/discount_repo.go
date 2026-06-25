package postgresgorm

import (
	"context"

	"gorm.io/gorm"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type DiscountRepo struct {
	db *gorm.DB
}

func NewDiscountRepo(db *gorm.DB) port.DiscountRepository {
	return &DiscountRepo{db: db}
}

func (r *DiscountRepo) Create(ctx context.Context, discount domain.Discount) (domain.Discount, error) {
	row := discountRowFromDomain(discount)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Discount{}, err
	}
	return r.FindById(ctx, discount.OrgId, discount.Id)
}

func (r *DiscountRepo) Update(ctx context.Context, discount domain.Discount) (domain.Discount, error) {
	row := discountRowFromDomain(discount)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.Discount{}, err
	}
	return r.FindById(ctx, discount.OrgId, discount.Id)
}

func (r *DiscountRepo) FindById(ctx context.Context, orgId, id string) (domain.Discount, error) {
	var row discountRow
	if err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.Discount{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *DiscountRepo) ActiveForSubscription(ctx context.Context, orgId, subscriptionId string) ([]domain.Discount, error) {
	return r.activeBy(ctx, orgId, "subscription_id = ?", subscriptionId)
}

// ActiveForOrder returns only order-level discounts (subscription_id IS NULL),
// so a subscription-targeted discount never leaks into a one-time order invoice.
func (r *DiscountRepo) ActiveForOrder(ctx context.Context, orgId, orderId string) ([]domain.Discount, error) {
	var rows []discountRow
	err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).
		Where("status = ?", domain.DiscountStatusActive).
		Where("order_id = ?", orderId).
		Where("subscription_id IS NULL").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return discountsToDomain(rows), nil
}

func (r *DiscountRepo) activeBy(ctx context.Context, orgId, where, arg string) ([]domain.Discount, error) {
	var rows []discountRow
	err := dbFromCtx(ctx, r.db).Scopes(OrgScope(orgId)).
		Where("status = ?", domain.DiscountStatusActive).
		Where(where, arg).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return discountsToDomain(rows), nil
}

func discountsToDomain(rows []discountRow) []domain.Discount {
	out := make([]domain.Discount, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out
}

func (r *DiscountRepo) CountByCoupon(ctx context.Context, orgId, couponId string) (int, error) {
	var n int64
	err := dbFromCtx(ctx, r.db).Model(&discountRow{}).
		Scopes(OrgScope(orgId)).Where("coupon_id = ?", couponId).Count(&n).Error
	return int(n), err
}

func (r *DiscountRepo) CountByCouponAndCustomer(ctx context.Context, orgId, couponId, customerId string) (int, error) {
	var n int64
	err := dbFromCtx(ctx, r.db).Model(&discountRow{}).
		Scopes(OrgScope(orgId)).Where("coupon_id = ? AND customer_id = ?", couponId, customerId).Count(&n).Error
	return int(n), err
}
