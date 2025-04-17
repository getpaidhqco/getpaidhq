package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type WebhookSubscriptionRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewWebhookSubscriptionRepository(primaryDb lib.Database, logger logger.Logger) repositories.WebhookSubscriptionRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return WebhookSubscriptionRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r WebhookSubscriptionRepository) Create(ctx context.Context, subscription entities.WebhookSubscription) (entities.WebhookSubscription, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO webhook_subscriptions (org_id, id, events, url, secret, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7)
			  RETURNING org_id, id, events, url, secret, created_at, updated_at`
	err := tx.QueryRow(ctx, query, subscription.OrgID, subscription.Id, subscription.Events, subscription.URL, subscription.Secret, subscription.CreatedAt, subscription.UpdatedAt).
		Scan(&subscription.OrgID, &subscription.Id, &subscription.Events, &subscription.URL, &subscription.Secret, &subscription.CreatedAt, &subscription.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to insert WebhookSubscription", err)
		return entities.WebhookSubscription{}, err
	}
	return subscription, nil
}

func (r WebhookSubscriptionRepository) GetByID(ctx context.Context, orgId string, id string) (entities.WebhookSubscription, error) {
	tx := r.getTransactionFromContext(ctx)

	var subscription entities.WebhookSubscription
	query := `SELECT org_id, id, events, url, secret, created_at, updated_at FROM webhook_subscriptions WHERE org_id=$1 AND id = $2`
	err := tx.QueryRow(ctx, query, orgId, id).
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

func (r WebhookSubscriptionRepository) FindByEvent(ctx context.Context, orgId string, event string) ([]entities.WebhookSubscription, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, events, url, secret, created_at, updated_at 
          FROM webhook_subscriptions 
          WHERE org_id = @org_id
           AND (@event = ANY(events) OR '*' = ANY(events))`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"event":  event,
	})
	if err != nil {
		r.logger.Error("failed to find WebhookSubscriptions by event", err)
		return nil, err
	}
	defer rows.Close()

	var subscriptions []entities.WebhookSubscription
	for rows.Next() {
		var subscription entities.WebhookSubscription
		err := rows.Scan(
			&subscription.OrgID,
			&subscription.Id,
			&subscription.Events,
			&subscription.URL,
			&subscription.Secret,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan WebhookSubscription", err)
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("rows error", err)
		return nil, err
	}

	return subscriptions, nil
}

func (r WebhookSubscriptionRepository) Update(ctx context.Context, subscription entities.WebhookSubscription) (entities.WebhookSubscription, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE webhook_subscriptions SET events = $1, url = $2, secret = $3, updated_at = $4 WHERE id = $5
			  RETURNING org_id, id, events, url, secret, created_at, updated_at`
	err := tx.QueryRow(ctx, query, subscription.Events, subscription.URL, subscription.Secret, subscription.UpdatedAt, subscription.Id).
		Scan(&subscription.OrgID, &subscription.Id, &subscription.Events, &subscription.URL, &subscription.Secret, &subscription.CreatedAt, &subscription.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to update WebhookSubscription", err)
		return entities.WebhookSubscription{}, err
	}
	return subscription, nil
}

func (r WebhookSubscriptionRepository) Delete(ctx context.Context, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM webhook_subscriptions WHERE id = $1`
	_, err := tx.Exec(ctx, query, id)
	if err != nil {
		r.logger.Error("failed to delete WebhookSubscription", err)
		return err
	}
	return nil
}
