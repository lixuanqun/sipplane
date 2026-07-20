package location

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements LocationStore with Redis + optional local memory cache (RFC 0005).
type RedisStore struct {
	Client     redis.Cmdable
	KeyPrefix  string
	Timeout    time.Duration
	LocalCache *MemoryStore
	CacheTTL   time.Duration
}

func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{
		Client:     client,
		KeyPrefix:  "sipplane:loc:",
		Timeout:    100 * time.Millisecond,
		LocalCache: NewMemoryStore(),
		CacheTTL:   5 * time.Second,
	}
}

func (s *RedisStore) key(aor string) string {
	return s.KeyPrefix + aor
}

func (s *RedisStore) Put(aor string, contacts []Contact, expires time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()
	if expires <= 0 || len(contacts) == 0 {
		_ = s.LocalCache.Delete(aor)
		return s.Client.Del(ctx, s.key(aor)).Err()
	}
	payload, err := json.Marshal(contacts)
	if err != nil {
		return err
	}
	if err := s.Client.Set(ctx, s.key(aor), payload, expires).Err(); err != nil {
		return err
	}
	cacheTTL := s.CacheTTL
	if expires < cacheTTL {
		cacheTTL = expires
	}
	return s.LocalCache.Put(aor, contacts, cacheTTL)
}

func (s *RedisStore) Get(aor string) ([]Contact, error) {
	if s.LocalCache != nil {
		if c, err := s.LocalCache.Get(aor); err == nil {
			return c, nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()
	val, err := s.Client.Get(ctx, s.key(aor)).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("location redis: %w", err)
	}
	var contacts []Contact
	if err := json.Unmarshal(val, &contacts); err != nil {
		return nil, err
	}
	if len(contacts) == 0 {
		return nil, ErrNotFound
	}
	ttl := s.CacheTTL
	_ = s.LocalCache.Put(aor, contacts, ttl)
	return contacts, nil
}

func (s *RedisStore) Delete(aor string) error {
	_ = s.LocalCache.Delete(aor)
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()
	return s.Client.Del(ctx, s.key(aor)).Err()
}

func (s *RedisStore) Count() int {
	// Approximate via local cache; full SCAN omitted for P3 MVP.
	return s.LocalCache.Count()
}
