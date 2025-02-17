package postgres

import (
	"context"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
	"time"
)

type IdempotencyKeyRepository struct {
	*lib.PgDatabase
	logger logger.Logger
}

func NewIdempotencyKeyRepository(database lib.Database, logger logger.Logger) repositories.IdempotencyKeyRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return IdempotencyKeyRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r IdempotencyKeyRepository) Exists(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := r.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM idempotency_keys WHERE id = $1 AND expires_at > NOW())", key).Scan(&exists)
	return exists, err
}

func (r IdempotencyKeyRepository) Create(ctx context.Context, key string, expiresAt time.Time) error {
	_, err := r.Pool.Exec(ctx, "INSERT INTO idempotency_keys ( id, expires_at,created_at,updated_at) VALUES ($1, $2,NOW(),NOW())", key, expiresAt)
	return err
}
