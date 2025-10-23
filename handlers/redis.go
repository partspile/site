package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/parts-pile/site/config"
	"github.com/redis/go-redis/v9"
)

var redisClient = redis.NewClient(&redis.Options{
	Addr:     config.RedisAddress,
	Password: config.RedisPassword,
})

const (
	keyUserValid = "user:valid:%d"
)

func redisSetUserValid(userID int) {
	ctx := context.Background()
	key := fmt.Sprintf(keyUserValid, userID)
	redisClient.Set(ctx, key, userID, time.Hour)
}

func redisUserInvalid(userID int) bool {
	ctx := context.Background()
	key := fmt.Sprintf(keyUserValid, userID)
	value, err := redisClient.Get(ctx, key).Int()
	return err != nil || value != userID
}

func redisSetUserInvalid(userID int) {
	ctx := context.Background()
	key := fmt.Sprintf(keyUserValid, userID)
	redisClient.Del(ctx, key)
}
