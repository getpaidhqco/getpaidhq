package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"

	"getpaidhq/internal/core/port"
)

// RateLimiter is a distributed implementation of port.RateLimiter backed by
// Redis. It uses go-redis/redis_rate, whose GCRA algorithm is evaluated
// atomically inside Redis via a Lua script, so the limit is enforced
// consistently across every API instance sharing the same Redis — unlike the
// in-memory limiter, whose budget is per process.
type RateLimiter struct {
	client  *redis.Client
	limiter *redis_rate.Limiter
	prefix  string
}

// NewRateLimiter builds a Redis-backed limiter on its own connection pool
// (kept separate from the cache pool so limiter traffic and cache traffic don't
// contend). Keys are namespaced under "ratelimit:" to avoid colliding with
// cache entries. The caller owns the returned limiter's lifecycle and should
// Close it on shutdown to release the connection pool.
func NewRateLimiter(addr, password string, db int) *RateLimiter {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RateLimiter{
		client:  rdb,
		limiter: redis_rate.NewLimiter(rdb),
		prefix:  "ratelimit:",
	}
}

// Close releases the limiter's Redis connection pool. It satisfies io.Closer so
// the application wiring can register it among the shutdown closers.
func (r *RateLimiter) Close() error {
	return r.client.Close()
}

// Allow consumes one token for key against an rps/burst budget. A transport or
// Redis error is returned to the caller (the HTTP middleware fails open on it,
// so a Redis blip degrades to "no limiting" rather than a full outage).
func (r *RateLimiter) Allow(ctx context.Context, key string, rps int, burst int) (port.RateLimitResult, error) {
	if burst <= 0 {
		burst = rps
		if burst < 1 {
			burst = 1
		}
	}

	res, err := r.limiter.Allow(ctx, r.prefix+key, redis_rate.Limit{
		Rate:   rps,
		Burst:  burst,
		Period: time.Second,
	})
	if err != nil {
		return port.RateLimitResult{}, err
	}

	return port.RateLimitResult{
		Allowed:    res.Allowed > 0,
		RetryAfter: res.RetryAfter,
	}, nil
}
