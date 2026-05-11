package port

import (
	"context"
	"time"
)

// CacheClient defines the interface for a cache adapter.
type CacheClient interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}
