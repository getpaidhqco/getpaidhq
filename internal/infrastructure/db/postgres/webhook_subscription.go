package postgres

import (
	"context"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type WebhookSubscriptionRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewWebhookSubscriptionRepository(database lib.Database, logger lib.Logger) repositories.WebhookSubscriptionRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return WebhookSubscriptionRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r WebhookSubscriptionRepository) Create(ctx context.Context, subscription entities.WebhookSubscription) (entities.WebhookSubscription, error) {
	query := `INSERT INTO webhook_subscriptions (org_id, id, events, url, secret, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7)
			  RETURNING org_id, id, events, url, secret, created_at, updated_at`
	err := r.Pool.QueryRow(ctx, query, subscription.OrgID, subscription.Id, subscription.Events, subscription.URL, subscription.Secret, subscription.CreatedAt, subscription.UpdatedAt).
		Scan(&subscription.OrgID, &subscription.Id, &subscription.Events, &subscription.URL, &subscription.Secret, &subscription.CreatedAt, &subscription.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to insert WebhookSubscription", err)
		return entities.WebhookSubscription{}, err
	}
	return subscription, nil
}

func (r WebhookSubscriptionRepository) GetByID(ctx context.Context, orgId string, id string) (entities.WebhookSubscription, error) {
	var subscription entities.WebhookSubscription
	query := `SELECT org_id, id, events, url, secret, created_at, updated_at FROM webhook_subscriptions WHERE org_id=$1 AND id = $2`
	err := r.Pool.QueryRow(ctx, query, orgId, id).
		Scan(
			&subscription.OrgID,
			&subscription.Id,
			&subscription.Events,
			&subscription.URL,
			&subscription.Secret,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		)
	if err != nil {
		r.logger.Error("failed to get WebhookSubscription by Id", err)
		return entities.WebhookSubscription{}, err
	}
	return subscription, nil
}

func (r WebhookSubscriptionRepository) Update(ctx context.Context, subscription entities.WebhookSubscription) (entities.WebhookSubscription, error) {
	query := `UPDATE webhook_subscriptions SET events = $1, url = $2, secret = $3, updated_at = $4 WHERE id = $5
			  RETURNING org_id, id, events, url, secret, created_at, updated_at`
	err := r.Pool.QueryRow(ctx, query, subscription.Events, subscription.URL, subscription.Secret, subscription.UpdatedAt, subscription.Id).
		Scan(&subscription.OrgID, &subscription.Id, &subscription.Events, &subscription.URL, &subscription.Secret, &subscription.CreatedAt, &subscription.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to update WebhookSubscription", err)
		return entities.WebhookSubscription{}, err
	}
	return subscription, nil
}

func (r WebhookSubscriptionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM webhook_subscriptions WHERE id = $1`
	_, err := r.Pool.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error("failed to delete WebhookSubscription", err)
		return err
	}
	return nil
}
