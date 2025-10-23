package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/parts-pile/site/config"
	"github.com/redis/go-redis/v9"
)

var Client = redis.NewClient(&redis.Options{
	Addr:         config.RedisAddress,
	Password:     config.RedisPassword,
	DialTimeout:  2 * time.Second, // How long to wait when establishing connection
	ReadTimeout:  1 * time.Second, // How long to wait for response
	WriteTimeout: 1 * time.Second, // How long to wait when sending data
})

const (
	keyUserValid   = "user:valid:%d"
	keyUserInvalid = "user:invalid:%d"
)

// SetUserValid marks a user as valid in Redis cache
func SetUserValid(userID int) {
	ctx := context.Background()
	key := fmt.Sprintf(keyUserValid, userID)
	Client.Set(ctx, key, userID, time.Hour)
}

// UserInvalid checks if a user is invalid (not in Redis cache or wrong value)
func UserInvalid(userID int) bool {
	ctx := context.Background()
	key := fmt.Sprintf(keyUserValid, userID)
	value, err := Client.Get(ctx, key).Int()
	return err != nil || value != userID
}

// SetUserInvalid marks a user as invalid in Redis cache
func SetUserInvalid(userID int) {
	ctx := context.Background()
	key := fmt.Sprintf(keyUserInvalid, userID)
	Client.Set(ctx, key, "1", time.Hour) // Keep for 1 hour
}

// IsUserInvalid checks if a user is marked as invalid
func IsUserInvalid(userID int) bool {
	ctx := context.Background()
	key := fmt.Sprintf(keyUserInvalid, userID)
	_, err := Client.Get(ctx, key).Result()
	return err == nil // If key exists, user is invalid
}

// StartHealthCheck starts a background goroutine that periodically checks Redis health
func StartHealthCheck() {
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
		defer ticker.Stop()

		log.Printf("[redis] Starting health check for Redis at %s", config.RedisAddress)

		for range ticker.C {
			ctx := context.Background()
			err := Client.Ping(ctx).Err()

			if err != nil {
				log.Printf("[redis] HEALTH CHECK FAILED - Redis server at %s is down: %v", config.RedisAddress, err)
			}
		}
	}()
}
