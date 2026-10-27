package loginlimit

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/kv"
	"github.com/redis/go-redis/v9"
)

const errTooMany = "Too many login attempts. Please try again later."

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func userKey(username string) string {
	return "login:user:" + normalizeUsername(username)
}

// Allow returns an error if this username is locked out from failures.
func Allow(username string) error {
	if config.AllowTestRegistration {
		return nil
	}
	c := kv.Client()
	if c == nil {
		return errors.New(errTooMany)
	}
	return allowRedis(c, username)
}

func allowRedis(c *redis.Client, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	n, err := c.Get(ctx, userKey(username)).Int()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return errors.New(errTooMany)
	}
	if n >= config.EffectiveLoginUserFailMax() {
		return errors.New(errTooMany)
	}
	return nil
}

// RecordFailure increments the per-username failure counter.
func RecordFailure(username string) {
	if config.AllowTestRegistration {
		return
	}
	c := kv.Client()
	if c == nil {
		return
	}
	recordFailureRedis(c, username)
}

func recordFailureRedis(c *redis.Client, username string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := userKey(username)
	n, err := c.Incr(ctx, key).Result()
	if err != nil {
		return
	}
	if n == 1 {
		_ = c.Expire(ctx, key, config.LoginUserFailExp).Err()
	}
}

// Clear removes the per-username failure counter after a successful login.
func Clear(username string) {
	c := kv.Client()
	if c == nil {
		return
	}
	clearRedis(c, username)
}

func clearRedis(c *redis.Client, username string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = c.Del(ctx, userKey(username)).Err()
}
