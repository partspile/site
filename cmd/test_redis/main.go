package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Get Redis connection details from environment variables
	redisAddress := os.Getenv("REDIS_ADDRESS")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	if redisAddress == "" {
		log.Fatal("REDIS_ADDRESS environment variable is required")
	}

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: redisPassword,
		DB:       0, // use default DB
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test connection
	fmt.Println("Testing Redis connection...")
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	fmt.Printf("Redis connection successful: %s\n", pong)

	// Test basic operations
	fmt.Println("\nTesting basic Redis operations...")

	// Set a key-value pair
	testKey := "test:key"
	testValue := "Hello Redis!"
	err = rdb.Set(ctx, testKey, testValue, 0).Err()
	if err != nil {
		log.Fatalf("Failed to set key: %v", err)
	}
	fmt.Printf("✓ Set key '%s' to '%s'\n", testKey, testValue)

	// Get the value back
	val, err := rdb.Get(ctx, testKey).Result()
	if err != nil {
		log.Fatalf("Failed to get key: %v", err)
	}
	fmt.Printf("✓ Retrieved key '%s': '%s'\n", testKey, val)

	// Test with expiration
	expireKey := "test:expire"
	expireValue := "This will expire"
	err = rdb.Set(ctx, expireKey, expireValue, 5*time.Second).Err()
	if err != nil {
		log.Fatalf("Failed to set key with expiration: %v", err)
	}
	fmt.Printf("✓ Set key '%s' with 5 second expiration\n", expireKey)

	// Check TTL
	ttl, err := rdb.TTL(ctx, expireKey).Result()
	if err != nil {
		log.Fatalf("Failed to get TTL: %v", err)
	}
	fmt.Printf("✓ TTL for '%s': %v\n", expireKey, ttl)

	// Test list operations
	listKey := "test:list"
	err = rdb.LPush(ctx, listKey, "item1", "item2", "item3").Err()
	if err != nil {
		log.Fatalf("Failed to push to list: %v", err)
	}
	fmt.Printf("✓ Pushed items to list '%s'\n", listKey)

	// Get list length
	length, err := rdb.LLen(ctx, listKey).Result()
	if err != nil {
		log.Fatalf("Failed to get list length: %v", err)
	}
	fmt.Printf("✓ List '%s' length: %d\n", listKey, length)

	// Pop from list
	item, err := rdb.RPop(ctx, listKey).Result()
	if err != nil {
		log.Fatalf("Failed to pop from list: %v", err)
	}
	fmt.Printf("✓ Popped from list '%s': '%s'\n", listKey, item)

	// Test hash operations
	hashKey := "test:hash"
	err = rdb.HSet(ctx, hashKey, "field1", "value1", "field2", "value2").Err()
	if err != nil {
		log.Fatalf("Failed to set hash: %v", err)
	}
	fmt.Printf("✓ Set hash '%s'\n", hashKey)

	// Get hash field
	hashVal, err := rdb.HGet(ctx, hashKey, "field1").Result()
	if err != nil {
		log.Fatalf("Failed to get hash field: %v", err)
	}
	fmt.Printf("✓ Hash field 'field1': '%s'\n", hashVal)

	// Test increment
	counterKey := "test:counter"
	err = rdb.Set(ctx, counterKey, 0, 0).Err()
	if err != nil {
		log.Fatalf("Failed to set counter: %v", err)
	}

	for i := 0; i < 3; i++ {
		val, err := rdb.Incr(ctx, counterKey).Result()
		if err != nil {
			log.Fatalf("Failed to increment counter: %v", err)
		}
		fmt.Printf("✓ Counter incremented to: %d\n", val)
	}

	// Clean up test keys
	fmt.Println("\nCleaning up test keys...")
	keys := []string{testKey, expireKey, listKey, hashKey, counterKey}
	for _, key := range keys {
		err = rdb.Del(ctx, key).Err()
		if err != nil {
			log.Printf("Warning: Failed to delete key '%s': %v", key, err)
		} else {
			fmt.Printf("✓ Deleted key '%s'\n", key)
		}
	}

	fmt.Println("\n✅ All Redis tests completed successfully!")
}
