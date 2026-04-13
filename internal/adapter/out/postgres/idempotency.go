package postgres

import (
	"context"
	"payloop/internal/core/port"
	"payloop/internal/lib"
	"time"
)

type IdempotencyKeyRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewIdempotencyKeyRepository(primaryDb lib.Database, logger port.Logger) port.IdempotencyKeyRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return IdempotencyKeyRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r IdempotencyKeyRepository) Exists(ctx context.Context, key string) (bool, error) {
	tx := r.getTransactionFromContext(ctx)

	var exists bool
	err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM idempotency_keys WHERE id = $1 AND expires_at > NOW())", key).Scan(&exists)
	return exists, err
}

func (r IdempotencyKeyRepository) Create(ctx context.Context, key string, expiresAt time.Time) error {
	tx := r.getTransactionFromContext(ctx)

	_, err := tx.Exec(ctx, "INSERT INTO idempotency_keys ( id, expires_at,created_at,updated_at) VALUES ($1, $2,NOW(),NOW())", key, expiresAt)
	return err
}
