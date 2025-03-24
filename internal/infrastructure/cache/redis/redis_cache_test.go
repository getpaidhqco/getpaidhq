package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRedisClient_Elasticache(t *testing.T) {
	addr := "localhost:6379"
	password := ""
	db := 0

	client := NewRedisClient(addr, password, db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Set(ctx, "test-key", "test-value", 1*time.Minute)
	assert.NoError(t, err)

	value, err := client.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.Equal(t, "test-value", value)

	err = client.Delete(ctx, "test-key")
	assert.NoError(t, err)
}
