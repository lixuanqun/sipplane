package location

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisStorePutGetDelete(t *testing.T) {
	addr := os.Getenv("SIPPLANE_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available at %s: %v", addr, err)
	}
	s := NewRedisStore(rdb)
	s.KeyPrefix = "sipplane:test:loc:"
	aor := "sip:redis-test@acme.example"
	_ = s.Delete(aor)

	err := s.Put(aor, []Contact{{
		URI: "sip:redis-test@10.0.0.1:5060", HostPort: "10.0.0.1:5060", Transport: "udp",
	}}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(aor)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].HostPort != "10.0.0.1:5060" {
		t.Fatalf("%+v", got)
	}
	s.LocalCache = NewMemoryStore()
	got2, err := s.Get(aor)
	if err != nil || len(got2) != 1 {
		t.Fatalf("redis hit failed: %v %+v", err, got2)
	}
	if err := s.Delete(aor); err != nil {
		t.Fatal(err)
	}
	s.LocalCache = NewMemoryStore()
	_, err = s.Get(aor)
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestRedisStoreExpiry(t *testing.T) {
	addr := os.Getenv("SIPPLANE_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available at %s: %v", addr, err)
	}
	s := NewRedisStore(rdb)
	s.KeyPrefix = "sipplane:test:exp:"
	s.Timeout = time.Second
	s.CacheTTL = 50 * time.Millisecond
	aor := "sip:redis-exp@acme.example"
	_ = s.Delete(aor)

	if err := s.Put(aor, []Contact{{
		URI: "sip:redis-exp@10.0.0.1:5060", HostPort: "10.0.0.1:5060", Transport: "udp",
	}}, 200*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(aor); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	s.LocalCache = NewMemoryStore() // drop any residual local entry
	_, err := s.Get(aor)
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound after expiry, got %v", err)
	}
}
