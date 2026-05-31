package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClock is a manually-advanced clock so token-bucket refill and TTL
// eviction are fully deterministic (no time.Sleep, no flakiness).
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newClock() *fakeClock {
	return &fakeClock{t: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// allow is a tiny helper that drops the context + error (this limiter never
// errors) so the bucket assertions read cleanly.
func allow(t *testing.T, m *RateLimiter, key string, rps, burst int) bool {
	t.Helper()
	res, err := m.Allow(context.Background(), key, rps, burst)
	require.NoError(t, err)
	return res.Allowed
}

func TestMemoryRateLimiter_AllowsBurstThenBlocks(t *testing.T) {
	t.Parallel()
	clock := newClock()
	m := NewRateLimiter(time.Minute)
	m.now = clock.now // freeze time: no refill during the burst

	// Burst capacity 3 → first three allowed, fourth blocked.
	for i := 0; i < 3; i++ {
		require.True(t, allow(t, m, "alice", 1, 3), "request %d within burst", i+1)
	}
	res, err := m.Allow(context.Background(), "alice", 1, 3)
	require.NoError(t, err)
	assert.False(t, res.Allowed)
	assert.Greater(t, res.RetryAfter, time.Duration(0), "a denial should hint when to retry")
}

func TestMemoryRateLimiter_RefillsOverTime(t *testing.T) {
	t.Parallel()
	clock := newClock()
	m := NewRateLimiter(time.Minute)
	m.now = clock.now

	require.True(t, allow(t, m, "alice", 1, 1))
	require.False(t, allow(t, m, "alice", 1, 1), "bucket empty at same instant")

	clock.advance(time.Second) // exactly one token refills at RPS=1
	require.True(t, allow(t, m, "alice", 1, 1))
	require.False(t, allow(t, m, "alice", 1, 1))
}

func TestMemoryRateLimiter_PerKeyIsolation(t *testing.T) {
	t.Parallel()
	clock := newClock()
	m := NewRateLimiter(time.Minute)
	m.now = clock.now

	require.True(t, allow(t, m, "alice", 1, 1))
	require.False(t, allow(t, m, "alice", 1, 1), "alice exhausted")
	require.True(t, allow(t, m, "bob", 1, 1), "bob has his own bucket")
}

func TestMemoryRateLimiter_BurstDefaultsToRPS(t *testing.T) {
	t.Parallel()
	clock := newClock()
	m := NewRateLimiter(time.Minute)
	m.now = clock.now

	// burst <= 0 → defaults to rps (2 here).
	require.True(t, allow(t, m, "alice", 2, 0))
	require.True(t, allow(t, m, "alice", 2, 0))
	require.False(t, allow(t, m, "alice", 2, 0))
}

func TestMemoryRateLimiter_EvictsIdleBucketsAfterTTL(t *testing.T) {
	t.Parallel()
	clock := newClock()
	m := NewRateLimiter(time.Minute)
	m.now = clock.now

	require.True(t, allow(t, m, "alice", 1, 1))
	require.False(t, allow(t, m, "alice", 1, 1))

	m.mu.Lock()
	require.Len(t, m.clients, 1)
	m.mu.Unlock()

	// Idle past TTL; a request from another key triggers the amortized sweep,
	// which evicts alice's stale bucket.
	clock.advance(2 * time.Minute)
	require.True(t, allow(t, m, "bob", 1, 1))

	m.mu.Lock()
	_, aliceThere := m.clients["alice"]
	m.mu.Unlock()
	assert.False(t, aliceThere, "idle bucket should be evicted after TTL")

	// Eviction means alice gets a fresh full bucket on her next request.
	require.True(t, allow(t, m, "alice", 1, 1))
}

// TestMemoryRateLimiter_Concurrent hammers a single key from many goroutines to
// surface races under `go test -race`. With a frozen clock and capacity N,
// exactly N concurrent requests are allowed.
func TestMemoryRateLimiter_Concurrent(t *testing.T) {
	t.Parallel()
	clock := newClock()
	m := NewRateLimiter(time.Minute)
	m.now = clock.now

	const burst = 50
	const goroutines = 200
	var (
		wg      sync.WaitGroup
		allowed int64
	)
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// Call Allow directly (not the require-based helper): testify's
			// require must only be invoked from the test goroutine. This
			// limiter never errors, so dropping the error here is safe.
			res, _ := m.Allow(context.Background(), "shared", 1, burst)
			if res.Allowed {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(burst), allowed, "exactly burst requests pass with a frozen clock")
}
