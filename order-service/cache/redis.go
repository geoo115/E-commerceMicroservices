package cache

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func InitRedis() *redis.Client {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "redis" // default to the service name if not set
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	addr := host + ":" + port

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // or read from env if needed
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", addr, err)
	}
	log.Printf("✅ Connected to Redis at %s", addr)
	return RedisClient
}
