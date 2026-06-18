package postgrespgx

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/port"
)

// IdempotencyKeyRepo is the pgx implementation of port.IdempotencyKeyRepository.
// The idempotency_keys table keys on the `id` column (the domain "key"); there
// is no separate `key` column, so the domain key is stored in `id`.
type IdempotencyKeyRepo struct {
	pool *pgxpool.Pool
}

func NewIdempotencyKeyRepo(pool *pgxpool.Pool) port.IdempotencyKeyRepository {
	return &IdempotencyKeyRepo{pool: pool}
}

// Claim atomically inserts the idempotency key iff no LIVE (non-expired) row
// already exists, and reports whether THIS call won.
//
// The whole claim is a single statement, so two concurrent retries of the same
// webhook can never both win: Postgres serialises the row-level conflict.
//
//   - No row for the key      → INSERT runs, RowsAffected=1 → claimed.
//   - Row exists but expired   → ON CONFLICT DO UPDATE fires because the WHERE
//     guard (expires_at < now()) holds, refreshing expires_at, RowsAffected=1
//     → claimed (a stale 24h-old replay is processed afresh, not dropped).
//   - Row exists and is live   → the DO UPDATE WHERE guard is false, the update
//     is skipped, RowsAffected=0 → NOT claimed (work already done).
//
// This mirrors the gorm adapter's observable behaviour (sweep-expired-then-
// upsert-on-conflict) in one round-trip. created_at relies on its column
// default; updated_at is NOT NULL with no default, so it is supplied and
// refreshed on the expired-claim path.
func (r *IdempotencyKeyRepo) Claim(ctx context.Context, key string, expiresAt time.Time) (bool, error) {
	q := dbFromCtx(ctx, r.pool)
	tag, err := q.Exec(ctx,
		`INSERT INTO idempotency_keys (id, expires_at, updated_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (id) DO UPDATE
		     SET expires_at = EXCLUDED.expires_at, updated_at = now()
		     WHERE idempotency_keys.expires_at < now()`,
		key, expiresAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Release deletes the idempotency row so the PSP's next retry can claim it
// afresh. Idempotent — deleting a missing key is not an error.
func (r *IdempotencyKeyRepo) Release(ctx context.Context, key string) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx, `DELETE FROM idempotency_keys WHERE id = $1`, key)
	return err
}
