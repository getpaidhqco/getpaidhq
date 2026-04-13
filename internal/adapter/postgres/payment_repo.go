package postgres

import (
	"context"

	"gorm.io/gorm"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

type PaymentRepo struct {
	db *gorm.DB
}

func NewPaymentRepo(db *gorm.DB) port.PaymentRepository {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) FindById(ctx context.Context, orgId string, id string) (domain.Payment, error) {
	var payment domain.Payment
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&payment).Error
	return payment, err
}

func (r *PaymentRepo) FindByPspId(ctx context.Context, orgId string, id string) (domain.Payment, error) {
	var payment domain.Payment
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("psp_id = ?", id).
		First(&payment).Error
	return payment, err
}

func (r *PaymentRepo) ListByPspId(ctx context.Context, psp domain.Gateway, pspId string) ([]domain.Payment, error) {
	var payments []domain.Payment
	err := r.db.WithContext(ctx).
		Where("psp = ? AND psp_id = ?", psp, pspId).
		Find(&payments).Error
	return payments, err
}

func (r *PaymentRepo) FindBySubscriptionId(ctx context.Context, orgId string, id string, p domain.Pagination) ([]domain.Payment, int, error) {
	var payments []domain.Payment
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Payment{}).
		Scopes(OrgScope(orgId)).
		Where("subscription_id = ?", id).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.db.WithContext(ctx).
		Scopes(OrgScope(orgId), Paginate(p)).
		Where("subscription_id = ?", id).
		Find(&payments).Error
	return payments, int(count), err
}

func (r *PaymentRepo) Create(ctx context.Context, entity domain.Payment) (domain.Payment, error) {
	err := r.db.WithContext(ctx).Create(&entity).Error
	if err != nil {
		return domain.Payment{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentRepo) Update(ctx context.Context, entity domain.Payment) (domain.Payment, error) {
	err := r.db.WithContext(ctx).Save(&entity).Error
	if err != nil {
		return domain.Payment{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentRepo) CreateRefund(ctx context.Context, refund domain.Refund) (domain.Refund, error) {
	err := r.db.WithContext(ctx).Create(&refund).Error
	if err != nil {
		return domain.Refund{}, err
	}
	var created domain.Refund
	err = r.db.WithContext(ctx).
		Where("id = ?", refund.Id).
		First(&created).Error
	return created, err
}
