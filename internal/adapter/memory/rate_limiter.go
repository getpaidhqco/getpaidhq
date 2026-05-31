package memory

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"getpaidhq/internal/core/port"
)

// rlClient is one key's token bucket plus its last-seen time (for TTL eviction).
type rlClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is an in-process implementation of port.RateLimiter backed by a
// per-key golang.org/x/time/rate token bucket. It is the fallback used when no
// distributed store (Redis) is configured.
//
// Caveat: limits are enforced PER INSTANCE. Behind N replicas the effective
// global rate is N×. Use the Redis adapter when a cluster-wide limit matters.
//
// Idle buckets are evicted after TTL to bound memory. All state is guarded by a
// single mutex and there is NO background goroutine — eviction happens lazily,
// amortized to at most once per TTL window — so the limiter is leak-free and
// deterministic under an injected clock in tests.
type RateLimiter struct {
	ttl time.Duration
	// now is the clock; swapped in tests for determinism. Defaults to time.Now.
	now func() time.Time

	mu        sync.Mutex
	clients   map[string]*rlClient
	lastSweep time.Time
}

// NewRateLimiter returns an in-memory rate limiter. Idle keys are forgotten
// after ttl; a non-positive ttl defaults to 10 minutes.
func NewRateLimiter(ttl time.Duration) *RateLimiter {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &RateLimiter{
		ttl:     ttl,
		now:     time.Now,
		clients: make(map[string]*rlClient),
	}
}

// Allow consumes one token for key, creating the bucket on first use with the
// supplied rps/burst. rps/burst are read when the bucket is first created for a
// key; in this app they are constant config, so that is sufficient. A
// non-positive burst defaults to max(1, rps). This implementation never errors.
func (m *RateLimiter) Allow(_ context.Context, key string, rps int, burst int) (port.RateLimitResult, error) {
	if burst <= 0 {
		burst = rps
		if burst < 1 {
			burst = 1
		}
	}

	now := m.now()
	m.mu.Lock()
	m.sweepLocked(now)
	c, ok := m.clients[key]
	if !ok {
		c = &rlClient{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
		m.clients[key] = c
	}
	c.lastSeen = now
	limiter := c.limiter
	m.mu.Unlock()

	// rate.Limiter is internally synchronized, so AllowN runs outside our
	// mutex to keep the critical section short. The injected clock makes the
	// accounting deterministic for tests.
	if limiter.AllowN(now, 1) {
		return port.RateLimitResult{Allowed: true}, nil
	}

	// Denied: next token refills in 1/rps seconds.
	retry := time.Second
	if rps > 0 {
		retry = time.Duration(float64(time.Second) / float64(rps))
	}
	return port.RateLimitResult{Allowed: false, RetryAfter: retry}, nil
}

// sweepLocked drops buckets idle longer than ttl. Caller holds m.mu. Sweeping
// is amortized to at most once per ttl window so the hot path stays O(1).
func (m *RateLimiter) sweepLocked(now time.Time) {
	if now.Sub(m.lastSweep) < m.ttl {
		return
	}
	m.lastSweep = now
	for k, c := range m.clients {
		if now.Sub(c.lastSeen) > m.ttl {
			delete(m.clients, k)
		}
	}
}
