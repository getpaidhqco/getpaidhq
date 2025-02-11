package repositories

import (
	"context"
	"time"
)

type IdempotencyKeyRepository interface {
	Exists(ctx context.Context, key string) (bool, error)
	Create(ctx context.Context, key string, expiresAt time.Time) error
}
