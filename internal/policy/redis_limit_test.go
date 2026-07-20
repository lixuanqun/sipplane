package policy

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisRateLimitShared(t *testing.T) {
	addr := os.Getenv("SIPPLANE_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6380"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available at %s: %v", addr, err)
	}

	key := "test-shared-" + time.Now().Format("150405.000")
	a := &RedisRateLimit{Client: rdb, KeyPrefix: "sipplane:rl:test:", CPS: 1, Burst: 1}
	b := &RedisRateLimit{Client: rdb, KeyPrefix: "sipplane:rl:test:", CPS: 1, Burst: 1}

	if !a.Allow(ctx, key) {
		t.Fatal("first allow")
	}
	if b.Allow(ctx, key) {
		t.Fatal("second instance should see shared deny")
	}
}

func TestRedisRateLimitFailClosed(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	rl := &RedisRateLimit{Client: rdb, CPS: 10, Burst: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if rl.Allow(ctx, "x") {
		t.Fatal("unreachable redis should deny")
	}
}
