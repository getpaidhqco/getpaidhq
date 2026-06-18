package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type WebhookSubscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewWebhookSubscriptionRepo(pool *pgxpool.Pool) port.WebhookSubscriptionRepository {
	return &WebhookSubscriptionRepo{pool: pool}
}

func (r *WebhookSubscriptionRepo) Create(ctx context.Context, subscription domain.WebhookSubscription) (domain.WebhookSubscription, error) {
	row := webhookSubscriptionRowFromDomain(subscription)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO webhook_subscriptions (`+webhookSubscriptionColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		row.OrgID, row.Id, row.Events, row.URL, row.Secret, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.WebhookSubscription{}, translateErr(err)
	}
	return r.GetByID(ctx, subscription.OrgID, subscription.Id)
}

func (r *WebhookSubscriptionRepo) GetByID(ctx context.Context, orgId string, id string) (domain.WebhookSubscription, error) {
	q := dbFromCtx(ctx, r.pool)
	var row webhookSubscriptionRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+webhookSubscriptionColumns+` FROM webhook_subscriptions WHERE org_id = $1 AND id = $2`, orgId, id)); err != nil {
		return domain.WebhookSubscription{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *WebhookSubscriptionRepo) FindByEvent(ctx context.Context, orgId string, event string) ([]domain.WebhookSubscription, error) {
	// Postgres array-containment against the native text[] `events` column:
	// `event = ANY(events)`. Mirrors the gorm adapter exactly.
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+webhookSubscriptionColumns+` FROM webhook_subscriptions WHERE org_id = $1 AND $2 = ANY(events)`, orgId, event)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *WebhookSubscriptionRepo) Update(ctx context.Context, subscription domain.WebhookSubscription) (domain.WebhookSubscription, error) {
	row := webhookSubscriptionRowFromDomain(subscription)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE webhook_subscriptions SET events=$3, url=$4, secret=$5, updated_at=$6 WHERE org_id=$1 AND id=$2`,
		row.OrgID, row.Id, row.Events, row.URL, row.Secret, row.UpdatedAt)
	if err != nil {
		return domain.WebhookSubscription{}, translateErr(err)
	}
	return r.GetByID(ctx, subscription.OrgID, subscription.Id)
}

func (r *WebhookSubscriptionRepo) Delete(ctx context.Context, id string) error {
	// Not org-scoped — mirrors the gorm adapter, which deletes by id alone.
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx, `DELETE FROM webhook_subscriptions WHERE id = $1`, id)
	return err
}

// collect drains rows into domain webhook subscriptions, closing rows.
func (r *WebhookSubscriptionRepo) collect(rows pgx.Rows) ([]domain.WebhookSubscription, error) {
	defer rows.Close()
	var out []domain.WebhookSubscription
	for rows.Next() {
		var row webhookSubscriptionRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
