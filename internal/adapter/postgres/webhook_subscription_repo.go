package postgres

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type WebhookSubscriptionRepo struct {
	db *gorm.DB
}

func NewWebhookSubscriptionRepo(db *gorm.DB) port.WebhookSubscriptionRepository {
	return &WebhookSubscriptionRepo{db: db}
}

func (r *WebhookSubscriptionRepo) Create(ctx context.Context, subscription domain.WebhookSubscription) (domain.WebhookSubscription, error) {
	row := webhookSubscriptionRowFromDomain(subscription)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.WebhookSubscription{}, err
	}
	return r.GetByID(ctx, subscription.OrgID, subscription.Id)
}

func (r *WebhookSubscriptionRepo) GetByID(ctx context.Context, orgId string, id string) (domain.WebhookSubscription, error) {
	var row webhookSubscriptionRow
	err := dbFromCtx(ctx, r.db).
		Where("org_id = ? AND id = ?", orgId, id).
		First(&row).Error
	if err != nil {
		return domain.WebhookSubscription{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *WebhookSubscriptionRepo) FindByEvent(ctx context.Context, orgId string, event string) ([]domain.WebhookSubscription, error) {
	var rows []webhookSubscriptionRow
	// Postgres array-containment: events column is a JSON-serialized array; ANY()
	// works against a text array dialect, so this query depends on the column
	// being typed as text[] at the database layer. Preserved exactly from the
	// pre-row-split implementation.
	err := dbFromCtx(ctx, r.db).
		Where("org_id = ? AND ? = ANY(events)", orgId, event).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.WebhookSubscription, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, nil
}

func (r *WebhookSubscriptionRepo) Update(ctx context.Context, subscription domain.WebhookSubscription) (domain.WebhookSubscription, error) {
	row := webhookSubscriptionRowFromDomain(subscription)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.WebhookSubscription{}, err
	}
	return r.GetByID(ctx, subscription.OrgID, subscription.Id)
}

func (r *WebhookSubscriptionRepo) Delete(ctx context.Context, id string) error {
	return dbFromCtx(ctx, r.db).
		Where("id = ?", id).
		Delete(&webhookSubscriptionRow{}).Error
}
