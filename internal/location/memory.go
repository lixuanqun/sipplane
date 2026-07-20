package location

import (
	"sync"
	"time"
)

// MemoryStore is an in-process LocationStore (P1).
type MemoryStore struct {
	mu   sync.RWMutex
	aors map[string][]Contact
}

// NewMemoryStore creates an empty in-memory location store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{aors: make(map[string][]Contact)}
}

func (s *MemoryStore) Put(aor string, contacts []Contact, expires time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if expires <= 0 || len(contacts) == 0 {
		delete(s.aors, aor)
		return nil
	}
	deadline := time.Now().Add(expires)
	copied := make([]Contact, len(contacts))
	for i, c := range contacts {
		c.ExpiresAt = deadline
		copied[i] = c
	}
	s.aors[aor] = copied
	return nil
}

func (s *MemoryStore) Get(aor string) ([]Contact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	contacts, ok := s.aors[aor]
	if !ok || len(contacts) == 0 {
		return nil, ErrNotFound
	}
	now := time.Now()
	alive := contacts[:0]
	for _, c := range contacts {
		if c.ExpiresAt.After(now) {
			alive = append(alive, c)
		}
	}
	if len(alive) == 0 {
		delete(s.aors, aor)
		return nil, ErrNotFound
	}
	s.aors[aor] = alive
	out := make([]Contact, len(alive))
	copy(out, alive)
	return out, nil
}

func (s *MemoryStore) Delete(aor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.aors, aor)
	return nil
}

func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	now := time.Now()
	for _, contacts := range s.aors {
		for _, c := range contacts {
			if c.ExpiresAt.After(now) {
				n++
				break
			}
		}
	}
	return n
}
