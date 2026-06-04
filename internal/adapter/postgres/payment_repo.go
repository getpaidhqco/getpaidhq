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
	var payment domain.Payment
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&payment).Error
	return payment, translateErr(err)
}

func (r *PaymentRepo) FindByPspId(ctx context.Context, orgId string, id string) (domain.Payment, error) {
	var payment domain.Payment
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("psp_id = ?", id).
		First(&payment).Error
	return payment, translateErr(err)
}

func (r *PaymentRepo) ListByPspId(ctx context.Context, psp domain.Gateway, pspId string) ([]domain.Payment, error) {
	var payments []domain.Payment
	err := dbFromCtx(ctx, r.db).
		Where("psp = ? AND psp_id = ?", psp, pspId).
		Find(&payments).Error
	return payments, err
}

func (r *PaymentRepo) FindBySubscriptionId(ctx context.Context, orgId string, id string, p domain.Pagination) ([]domain.Payment, int, error) {
	var payments []domain.Payment
	var count int64
	err := dbFromCtx(ctx, r.db).Model(&domain.Payment{}).
		Scopes(OrgScope(orgId)).
		Where("subscription_id = ?", id).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Where("subscription_id = ?", id).
		Find(&payments).Error
	return payments, int(count), err
}

func (r *PaymentRepo) Create(ctx context.Context, entity domain.Payment) (domain.Payment, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Payment{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentRepo) Update(ctx context.Context, entity domain.Payment) (domain.Payment, error) {
	err := dbFromCtx(ctx, r.db).Save(&entity).Error
	if err != nil {
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
