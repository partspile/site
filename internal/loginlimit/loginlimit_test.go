package loginlimit

import (
	"context"
	"reflect"
	"testing"

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

func disableTestRegistration(t *testing.T) {
	t.Helper()
	orig := config.AllowTestRegistration
	reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(false)
	t.Cleanup(func() {
		reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(orig)
	})
}

func TestAllowUnderCap(t *testing.T) {
	setupTestRedis(t)
	disableTestRegistration(t)

	if err := Allow("alice"); err != nil {
		t.Fatalf("allow with no failures: %v", err)
	}
	RecordFailure("alice")
	if err := Allow("alice"); err != nil {
		t.Fatalf("allow after one failure: %v", err)
	}
}

func TestAllowAtCap(t *testing.T) {
	setupTestRedis(t)
	disableTestRegistration(t)

	for i := 0; i < config.LoginUserFailMax; i++ {
		RecordFailure("bob")
	}
	if err := Allow("bob"); err == nil {
		t.Fatal("expected lockout at failure cap")
	}
}

func TestClearResetsFailures(t *testing.T) {
	setupTestRedis(t)
	disableTestRegistration(t)

	for i := 0; i < config.LoginUserFailMax; i++ {
		RecordFailure("carol")
	}
	Clear("carol")
	if err := Allow("carol"); err != nil {
		t.Fatalf("allow after clear: %v", err)
	}
}

func TestNormalizeCaseFold(t *testing.T) {
	setupTestRedis(t)
	disableTestRegistration(t)

	for i := 0; i < config.LoginUserFailMax; i++ {
		RecordFailure("Dave")
	}
	if err := Allow("dave"); err == nil {
		t.Fatal("expected lockout to apply across case")
	}
}

func TestSkippedWhenTestRegistration(t *testing.T) {
	orig := config.AllowTestRegistration
	reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(true)
	t.Cleanup(func() {
		reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(orig)
	})

	if err := Allow("anyone"); err != nil {
		t.Fatalf("allow should no-op in test mode: %v", err)
	}
	RecordFailure("anyone")
}
