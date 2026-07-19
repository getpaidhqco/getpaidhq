package postgrespgx

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type OutboxRepo struct {
	pool *pgxpool.Pool
}

func NewOutboxRepo(pool *pgxpool.Pool) port.OutboxRepository {
	return &OutboxRepo{pool: pool}
}

func (r *OutboxRepo) Create(ctx context.Context, ev domain.OutboxEvent) error {
	q := dbFromCtx(ctx, r.pool)
	var lastError *string
	if ev.LastError != "" {
		lastError = &ev.LastError
	}
	_, err := q.Exec(ctx,
		`INSERT INTO outbox_events (event_id, org_id, topic, payload, attempts, next_attempt_at, last_error, published_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		ev.EventId, ev.OrgId, ev.Topic, ev.Payload, ev.Attempts, ev.NextAttemptAt, lastError, ev.PublishedAt, ev.CreatedAt)
	return err
}

func (r *OutboxRepo) ClaimPending(ctx context.Context, limit int, maxAttempts int, now time.Time) ([]domain.OutboxEvent, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+outboxEventColumns+` FROM outbox_events
		 WHERE published_at IS NULL AND attempts < $1
		   AND (next_attempt_at IS NULL OR next_attempt_at <= $2)
		 ORDER BY id LIMIT $3
		 FOR UPDATE SKIP LOCKED`,
		maxAttempts, now, limit)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id int64, at time.Time) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx, `UPDATE outbox_events SET published_at = $1 WHERE id = $2`, at, id)
	return err
}

func (r *OutboxRepo) RecordFailure(ctx context.Context, id int64, lastError string, nextAttemptAt time.Time) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE outbox_events SET attempts = attempts + 1, last_error = $1, next_attempt_at = $2 WHERE id = $3`,
		lastError, nextAttemptAt, id)
	return err
}

func (r *OutboxRepo) PurgePublished(ctx context.Context, olderThan time.Time) (int64, error) {
	q := dbFromCtx(ctx, r.pool)
	tag, err := q.Exec(ctx,
		`DELETE FROM outbox_events WHERE published_at IS NOT NULL AND published_at < $1`, olderThan)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *OutboxRepo) collect(rows pgx.Rows) ([]domain.OutboxEvent, error) {
	defer rows.Close()
	out := []domain.OutboxEvent{}
	for rows.Next() {
		var row outboxEventRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
