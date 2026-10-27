package kv

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberredis "github.com/gofiber/storage/redis/v3"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
	"github.com/redis/go-redis/v9"
)

var (
	client  *redis.Client
	storage fiber.Storage
)

// Init connects to REDIS_URL (required).
func Init() error {
	url := config.RedisURL
	if url == "" {
		return fmt.Errorf("REDIS_URL is required")
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("parse REDIS_URL: %w", err)
	}

	c := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return fmt.Errorf("redis ping: %w", err)
	}

	setClient(c)
	logger.Info("Redis connected", "addr", opts.Addr)
	return nil
}

// InitWithClient wires an existing client (tests).
func InitWithClient(c *redis.Client) {
	setClient(c)
}

func setClient(c *redis.Client) {
	client = c
	storage = fiberredis.NewFromConnection(c)
}

// Client returns the shared Redis client.
func Client() *redis.Client {
	return client
}

// Storage returns Fiber storage for limiters.
func Storage() fiber.Storage {
	return storage
}

// Close releases the Redis client.
func Close() error {
	if client == nil {
		return nil
	}
	err := client.Close()
	client = nil
	storage = nil
	return err
}
