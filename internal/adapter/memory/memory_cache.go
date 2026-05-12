package memory

import (
	"context"
	"errors"
	"getpaidhq/internal/core/port"
	"time"
)

// InMemoryCache is a concrete implementation of the Cache interface using an in-memory map.
type InMemoryCache struct {
	store map[string]string
}

// NewInMemoryCache creates a new in-memory cache.
func NewInMemoryCache() port.CacheClient {
	return &InMemoryCache{store: make(map[string]string)}
}

// Set sets a key-value pair in the in-memory cache with an expiration time (ignored in this implementation).
func (m *InMemoryCache) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	m.store[key] = value.(string)
	return nil
}

// Get gets the value of a key from the in-memory cache.
func (m *InMemoryCache) Get(ctx context.Context, key string) (string, error) {
	value, ok := m.store[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return value, nil
}

// Delete deletes a key from the in-memory cache.
func (m *InMemoryCache) Delete(ctx context.Context, key string) error {
	delete(m.store, key)
	return nil
}
