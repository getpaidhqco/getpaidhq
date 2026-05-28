//go:build integration

package redis

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRedisClient_Elasticache exercises the real Redis round-trip. It is
// gated behind `//go:build integration` so a plain `go test ./...` on a machine
// without Redis stays green, and additionally skips unless REDIS_ADDR is set
// (mirrors the SQS_QUEUE_URL skip pattern). Run with:
//
//	REDIS_ADDR=localhost:6379 go test -tags=integration ./internal/adapter/redis/...
func TestNewRedisClient_Elasticache(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR not set; skipping Redis integration test")
	}

	client := NewRedisClient(addr, os.Getenv("REDIS_PASSWORD"), 0)
	ctx := t.Context()

	// Unique key per run so concurrent jobs / -count=N against a shared Redis
	// don't collide on each other's value.
	key := fmt.Sprintf("test-key-%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = client.Delete(ctx, key) })

	require.NoError(t, client.Set(ctx, key, "test-value", 1*time.Minute))

	value, err := client.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, "test-value", value)

	assert.NoError(t, client.Delete(ctx, key))
}
