package postgres

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type PaymentRepo struct {
	db *gorm.DB
}

func NewPaymentRepo(db *gorm.DB) port.PaymentRepository {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) FindById(ctx context.Context, orgId string, id string) (domain.Payment, error) {
	var row paymentRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Payment{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *PaymentRepo) FindByPspId(ctx context.Context, orgId string, id string) (domain.Payment, error) {
	var row paymentRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("psp_id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Payment{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *PaymentRepo) ListByPspId(ctx context.Context, psp domain.Gateway, pspId string) ([]domain.Payment, error) {
	var rows []paymentRow
	err := dbFromCtx(ctx, r.db).
		Where("psp = ? AND psp_id = ?", psp, pspId).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return paymentRowsToDomain(rows), nil
}

func (r *PaymentRepo) FindBySubscriptionId(ctx context.Context, orgId string, id string, p domain.Pagination) ([]domain.Payment, int, error) {
	var rows []paymentRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&paymentRow{}).
		Scopes(OrgScope(orgId)).
		Where("subscription_id = ?", id).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Where("subscription_id = ?", id).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return paymentRowsToDomain(rows), int(count), nil
}

func (r *PaymentRepo) Create(ctx context.Context, entity domain.Payment) (domain.Payment, error) {
	row := paymentRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Payment{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentRepo) Update(ctx context.Context, entity domain.Payment) (domain.Payment, error) {
	row := paymentRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.Payment{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentRepo) CreateRefund(ctx context.Context, refund domain.Refund) (domain.Refund, error) {
	row := refundRowFromDomain(refund)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Refund{}, err
	}
	var created refundRow
	err := dbFromCtx(ctx, r.db).
		Where("id = ?", refund.Id).
		First(&created).Error
	if err != nil {
		return domain.Refund{}, translateErr(err)
	}
	return created.toDomain(), nil
}
