package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
	"payloop/internal/application/lib/cache"
	"time"
)

// RedisClient is a concrete implementation of the Cache interface using Redis.
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client.
func NewRedisClient(addr string, password string, db int) cache.CacheClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisClient{client: rdb}
}

// Set sets a key-value pair in Redis with an expiration time.
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get gets the value of a key from Redis.
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Delete deletes a key from Redis.
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
