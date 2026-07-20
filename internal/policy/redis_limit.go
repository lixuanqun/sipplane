package policy

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimit is a shared token-bucket limiter (multi-instance).
// Redis errors fail closed (deny), aligned with RFC 0005 abuse posture.
type RedisRateLimit struct {
	Client    redis.Cmdable
	KeyPrefix string
	CPS       float64
	Burst     int
}

const redisLimitScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local data = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])
if tokens == nil then
  tokens = burst
end
if ts == nil then
  ts = now
end
local elapsed = now - ts
if elapsed < 0 then
  elapsed = 0
end
tokens = tokens + elapsed * rate
if tokens > burst then
  tokens = burst
end
local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end
redis.call('HMSET', key, 'tokens', tokens, 'ts', now)
redis.call('EXPIRE', key, 120)
return allowed
`

func (r *RedisRateLimit) Allow(ctx context.Context, key string) bool {
	if r == nil || r.Client == nil || r.CPS <= 0 {
		return true
	}
	prefix := r.KeyPrefix
	if prefix == "" {
		prefix = "sipplane:rl:"
	}
	burst := r.Burst
	if burst <= 0 {
		if r.CPS < 1 {
			burst = 1
		} else {
			burst = int(r.CPS)
		}
	}
	now := float64(time.Now().UnixNano()) / 1e9
	res, err := r.Client.Eval(ctx, redisLimitScript, []string{prefix + key}, r.CPS, burst, now).Int()
	if err != nil {
		return false
	}
	return res == 1
}
