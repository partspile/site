package otplimit

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/kv"
	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	c := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	if err := c.Ping(context.Background()).Err(); err != nil {
		mr.Close()
		t.Fatalf("redis ping: %v", err)
	}
	kv.InitWithClient(c)
	t.Cleanup(func() {
		_ = kv.Close()
		mr.Close()
	})
	return mr
}

func TestAllowStartCooldown(t *testing.T) {
	setupTestRedis(t)
	origAllow := config.AllowTestRegistration
	reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(false)
	t.Cleanup(func() {
		reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(origAllow)
	})

	phone := "+15559875001"
	if err := AllowStart(phone); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if err := AllowStart(phone); err == nil {
		t.Fatal("second start within interval should fail")
	}
}

func TestAllowStartHourlyCap(t *testing.T) {
	setupTestRedis(t)
	origAllow := config.AllowTestRegistration
	reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(false)
	t.Cleanup(func() {
		reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(origAllow)
	})

	phone := "+15559875002"
	ctx := context.Background()
	c := kv.Client()
	for i := 0; i < config.OTPStartMaxPerHour; i++ {
		if err := c.Incr(ctx, "otp:hr:"+phone).Err(); err != nil {
			t.Fatalf("seed hour counter: %v", err)
		}
	}
	if err := c.Expire(ctx, "otp:hr:"+phone, time.Hour).Err(); err != nil {
		t.Fatalf("seed hour ttl: %v", err)
	}

	if err := AllowStart(phone); err == nil {
		t.Fatal("hourly cap should block")
	}
}

func TestAllowStartSkippedWhenTestRegistration(t *testing.T) {
	origAllow := config.AllowTestRegistration
	reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(true)
	t.Cleanup(func() {
		reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(origAllow)
	})
	for i := 0; i < 20; i++ {
		if err := AllowStart("+15559875003"); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
}
