package postgres

import (
	"context"
	"time"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type PaymentMethodRepo struct {
	db *gorm.DB
}

func NewPaymentMethodRepo(db *gorm.DB) port.PaymentMethodRepository {
	return &PaymentMethodRepo{db: db}
}

func (r *PaymentMethodRepo) FindById(ctx context.Context, orgId string, id string) (domain.PaymentMethod, error) {
	var pm domain.PaymentMethod
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&pm).Error
	return pm, err
}

func (r *PaymentMethodRepo) Create(ctx context.Context, entity domain.PaymentMethod) (domain.PaymentMethod, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.PaymentMethod{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentMethodRepo) Update(ctx context.Context, entity domain.PaymentMethod) (domain.PaymentMethod, error) {
	err := dbFromCtx(ctx, r.db).Save(&entity).Error
	if err != nil {
		return domain.PaymentMethod{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentMethodRepo) FindExpiringPaymentMethods(ctx context.Context, expiry time.Time) ([]domain.PaymentMethod, error) {
	var methods []domain.PaymentMethod
	err := dbFromCtx(ctx, r.db).
		Where("expire_at <= ? AND expire_at > ?", expiry, time.Time{}).
		Find(&methods).Error
	return methods, err
}
