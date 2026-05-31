package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// redisAddrForTest returns the address of a live Redis to test against, or ""
// to signal "skip". It honors REDIS_TEST_ADDR and otherwise probes the
// conventional localhost:6379 so the test runs automatically when a dev Redis
// is up, but never fails CI when one isn't (Redis is not in the local stack).
func redisAddrForTest(t *testing.T) string {
	t.Helper()
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	probe := redis.NewClient(&redis.Options{Addr: addr})
	defer probe.Close()
	if err := probe.Ping(ctx).Err(); err != nil {
		t.Skipf("no reachable Redis at %s (set REDIS_TEST_ADDR to run): %v", addr, err)
	}
	return addr
}

// uniqueKey keeps test runs independent of each other and of any real data by
// namespacing on the test name. A leading flush of the key avoids contamination
// from a previous interrupted run.
func freshLimiter(t *testing.T, addr string) (*RateLimiter, func()) {
	t.Helper()
	rl := NewRateLimiter(addr, "", 0)
	clean := func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		c := redis.NewClient(&redis.Options{Addr: addr})
		defer c.Close()
		c.Del(ctx, rl.prefix+t.Name())
	}
	clean()
	return rl, clean
}

func TestRedisRateLimiter_AllowsBurstThenBlocks(t *testing.T) {
	addr := redisAddrForTest(t)
	rl, clean := freshLimiter(t, addr)
	defer clean()

	ctx := context.Background()
	key := t.Name()

	// GCRA with Rate=1/s, Burst=3 admits an initial burst of 3.
	for i := 0; i < 3; i++ {
		res, err := rl.Allow(ctx, key, 1, 3)
		require.NoError(t, err)
		require.True(t, res.Allowed, "request %d should be within burst", i+1)
	}

	res, err := rl.Allow(ctx, key, 1, 3)
	require.NoError(t, err)
	assert.False(t, res.Allowed, "burst exhausted ⇒ blocked")
	assert.Greater(t, res.RetryAfter, time.Duration(0), "blocked result carries a positive RetryAfter")
}

func TestRedisRateLimiter_KeyIsolation(t *testing.T) {
	addr := redisAddrForTest(t)
	rl, clean := freshLimiter(t, addr)
	defer clean()

	ctx := context.Background()

	// Exhaust key A (burst 1).
	resA, err := rl.Allow(ctx, t.Name()+":A", 1, 1)
	require.NoError(t, err)
	require.True(t, resA.Allowed)
	resA2, err := rl.Allow(ctx, t.Name()+":A", 1, 1)
	require.NoError(t, err)
	require.False(t, resA2.Allowed)

	// Key B is unaffected.
	resB, err := rl.Allow(ctx, t.Name()+":B", 1, 1)
	require.NoError(t, err)
	assert.True(t, resB.Allowed, "a different key has an independent budget")

	// Cleanup B's key too.
	defer func() {
		c := redis.NewClient(&redis.Options{Addr: addr})
		defer c.Close()
		c.Del(context.Background(), rl.prefix+t.Name()+":A", rl.prefix+t.Name()+":B")
	}()
}
