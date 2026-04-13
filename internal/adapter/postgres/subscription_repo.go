package postgres

import (
	"context"

	"gorm.io/gorm"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
)

type SubscriptionRepo struct {
	db *gorm.DB
}

func NewSubscriptionRepo(db *gorm.DB) port.SubscriptionRepository {
	return &SubscriptionRepo{db: db}
}

func (r *SubscriptionRepo) FindById(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	var sub domain.Subscription
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Preload("Customer").
		First(&sub).Error
	return sub, err
}

func (r *SubscriptionRepo) Create(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	err := r.db.WithContext(ctx).Create(&entity).Error
	if err != nil {
		return domain.Subscription{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) Update(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	err := r.db.WithContext(ctx).Save(&entity).Error
	if err != nil {
		return domain.Subscription{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *SubscriptionRepo) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
	var subs []domain.Subscription
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("order_id = ?", orderId).
		Find(&subs).Error
	return subs, err
}

func (r *SubscriptionRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Subscription, int, error) {
	var subs []domain.Subscription
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Subscription{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.db.WithContext(ctx).
		Scopes(OrgScope(orgId), Paginate(p)).
		Preload("Customer").
		Find(&subs).Error
	return subs, int(count), err
}
