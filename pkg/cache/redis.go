// Package cache opens Redis connections.
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/geoo115/E-commerceMicroservices/pkg/config"
)

// OpenFromEnv connects to the Redis instance described by REDIS_ADDR and REDIS_PASSWORD.
func OpenFromEnv(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.String("REDIS_ADDR", "localhost:6379"),
		Password: config.String("REDIS_PASSWORD", ""),
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to redis: %w", err)
	}
	return client, nil
}
