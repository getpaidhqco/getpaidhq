package cache

import (
	"context"
	"time"
)

// Cache is an interface for a generic cache client.
type CacheClient interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}
