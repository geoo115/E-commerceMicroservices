package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is a minimal byte cache used for cache-aside reads. Implementations
// must treat failures as cache misses: the database stays the source of truth.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration)
	Delete(ctx context.Context, keys ...string)
}

// Redis implements Cache on top of a Redis client.
type Redis struct {
	Client *redis.Client
}

// Get implements Cache.
func (r Redis) Get(ctx context.Context, key string) ([]byte, bool) {
	b, err := r.Client.Get(ctx, key).Bytes()
	return b, err == nil
}

// Set implements Cache.
func (r Redis) Set(ctx context.Context, key string, value []byte, ttl time.Duration) {
	r.Client.Set(ctx, key, value, ttl)
}

// Delete implements Cache.
func (r Redis) Delete(ctx context.Context, keys ...string) {
	r.Client.Del(ctx, keys...)
}

// Noop is a Cache that never stores anything. Useful in tests.
type Noop struct{}

// Get implements Cache.
func (Noop) Get(context.Context, string) ([]byte, bool) { return nil, false }

// Set implements Cache.
func (Noop) Set(context.Context, string, []byte, time.Duration) {}

// Delete implements Cache.
func (Noop) Delete(context.Context, ...string) {}
