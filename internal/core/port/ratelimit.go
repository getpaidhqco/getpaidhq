package port

import (
	"context"
	"time"
)

// RateLimitResult is the decision for a single rate-limit check.
type RateLimitResult struct {
	// Allowed reports whether the request may proceed.
	Allowed bool
	// RetryAfter is how long the caller should wait before retrying when
	// Allowed is false. Zero when allowed (or unknown).
	RetryAfter time.Duration
}

// RateLimiter decides whether a request identified by key may proceed under a
// per-key budget of rps requests/second with a burst capacity (token-bucket
// semantics). Implementations may be in-memory (per process instance) or
// distributed (shared across instances via Redis).
//
// Each call consumes one unit from the key's budget when allowed. A non-nil
// error means the decision could not be made (e.g. the backing store is
// unreachable); callers decide the policy (the HTTP middleware fails open).
type RateLimiter interface {
	Allow(ctx context.Context, key string, rps int, burst int) (RateLimitResult, error)
}
