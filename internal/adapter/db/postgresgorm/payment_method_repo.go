package postgresgorm

import (
	"context"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type PaymentMethodRepo struct {
	db *gorm.DB
}

func NewPaymentMethodRepo(db *gorm.DB) port.PaymentMethodRepository {
	return &PaymentMethodRepo{db: db}
}

func (r *PaymentMethodRepo) FindById(ctx context.Context, orgId string, id string) (domain.PaymentMethod, error) {
	var row paymentMethodRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.PaymentMethod{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *PaymentMethodRepo) Create(ctx context.Context, entity domain.PaymentMethod) (domain.PaymentMethod, error) {
	row := paymentMethodRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.PaymentMethod{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentMethodRepo) Update(ctx context.Context, entity domain.PaymentMethod) (domain.PaymentMethod, error) {
	row := paymentMethodRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.PaymentMethod{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *PaymentMethodRepo) FindExpiringPaymentMethods(ctx context.Context, expiry time.Time) ([]domain.PaymentMethod, error) {
	var rows []paymentMethodRow
	err := dbFromCtx(ctx, r.db).
		Where("expire_at <= ? AND expire_at > ?", expiry, time.Time{}).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return paymentMethodRowsToDomain(rows), nil
}
