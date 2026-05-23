package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type WebhookSubscriptionRepo struct {
	db *gorm.DB
}

func NewWebhookSubscriptionRepo(db *gorm.DB) port.WebhookSubscriptionRepository {
	return &WebhookSubscriptionRepo{db: db}
}

func (r *WebhookSubscriptionRepo) Create(ctx context.Context, subscription domain.WebhookSubscription) (domain.WebhookSubscription, error) {
	err := dbFromCtx(ctx, r.db).Create(&subscription).Error
	if err != nil {
		return domain.WebhookSubscription{}, err
	}
	return r.GetByID(ctx, subscription.OrgID, subscription.Id)
}

func (r *WebhookSubscriptionRepo) GetByID(ctx context.Context, orgId string, id string) (domain.WebhookSubscription, error) {
	var ws domain.WebhookSubscription
	err := dbFromCtx(ctx, r.db).
		Where("org_id = ? AND id = ?", orgId, id).
		First(&ws).Error
	return ws, err
}

func (r *WebhookSubscriptionRepo) FindByEvent(ctx context.Context, orgId string, event string) ([]domain.WebhookSubscription, error) {
	var subs []domain.WebhookSubscription
	// Use a raw query to find webhook subscriptions where the events array contains the given event.
	// This assumes a PostgreSQL array column or a JSON column for events.
	err := dbFromCtx(ctx, r.db).
		Where("org_id = ? AND ? = ANY(events)", orgId, event).
		Find(&subs).Error
	return subs, err
}

func (r *WebhookSubscriptionRepo) Update(ctx context.Context, subscription domain.WebhookSubscription) (domain.WebhookSubscription, error) {
	err := dbFromCtx(ctx, r.db).Save(&subscription).Error
	if err != nil {
		return domain.WebhookSubscription{}, err
	}
	return r.GetByID(ctx, subscription.OrgID, subscription.Id)
}

func (r *WebhookSubscriptionRepo) Delete(ctx context.Context, id string) error {
	return dbFromCtx(ctx, r.db).
		Where("id = ?", id).
		Delete(&domain.WebhookSubscription{}).Error
}
