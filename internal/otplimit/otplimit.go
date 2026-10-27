package otplimit

import (
	"context"
	"fmt"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/kv"
	"github.com/redis/go-redis/v9"
)

var (
	// allowStartScript: KEYS[1]=cooldown KEYS[2]=hour counter
	// ARGV[1]=cooldown TTL sec ARGV[2]=hour TTL sec ARGV[3]=max/hour
	// Returns: 1 ok | -1 cooldown | -2 hourly cap
	allowStartScript = redis.NewScript(`
local cd = KEYS[1]
local hr = KEYS[2]
local cdTTL = tonumber(ARGV[1])
local hrTTL = tonumber(ARGV[2])
local maxHr = tonumber(ARGV[3])
if redis.call('EXISTS', cd) == 1 then
  return -1
end
local n = redis.call('INCR', hr)
if n == 1 then
  redis.call('EXPIRE', hr, hrTTL)
end
if n > maxHr then
  redis.call('DECR', hr)
  return -2
end
redis.call('SET', cd, '1', 'EX', cdTTL)
return 1
`)
)

// AllowStart returns an error if this phone may not start another OTP yet.
// Limits are stored in Redis and shared across instances.
func AllowStart(phoneE64 string) error {
	if config.AllowTestRegistration {
		return nil
	}
	c := kv.Client()
	if c == nil {
		return fmt.Errorf("verification rate limit unavailable, try again")
	}
	return allowStartRedis(c, phoneE64)
}

func allowStartRedis(c *redis.Client, phoneE64 string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cdKey := "otp:cd:" + phoneE64
	hrKey := "otp:hr:" + phoneE64
	cdTTL := int(config.OTPStartMinInterval.Seconds())
	if cdTTL < 1 {
		cdTTL = 1
	}

	res, err := allowStartScript.Run(ctx, c,
		[]string{cdKey, hrKey},
		cdTTL,
		3600,
		config.OTPStartMaxPerHour,
	).Int()
	if err != nil {
		return fmt.Errorf("verification rate limit unavailable, try again")
	}
	switch res {
	case 1:
		return nil
	case -1:
		ttl, _ := c.TTL(ctx, cdKey).Result()
		wait := int(ttl.Seconds()) + 1
		if wait < 1 {
			wait = int(config.OTPStartMinInterval.Seconds())
		}
		return fmt.Errorf(
			"please wait %d seconds before requesting another code", wait)
	case -2:
		return fmt.Errorf(
			"too many verification codes requested for this number. " +
				"Please try again later.")
	default:
		return fmt.Errorf("verification rate limit unavailable, try again")
	}
}
