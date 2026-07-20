package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sipplane/sipplane/internal/resources"
)

var (
	ErrNotFound      = errors.New("store: not found")
	ErrConflict      = errors.New("store: conflict")
	ErrInvalid       = errors.New("store: invalid")
	ErrValidation    = errors.New("store: validation failed")
)

// Resource is a stored config object.
type Resource struct {
	Kind   string          `json:"kind"`
	Tenant string          `json:"tenant"`
	Name   string          `json:"name"`
	Spec   json.RawMessage `json:"spec"`
}

// AuditEntry records who changed what.
type AuditEntry struct {
	Revision int64
	Action   string
	Actor    string
	Kind     string
	Tenant   string
	Name     string
	At       time.Time
}

// Store is the control-plane config store (RFC 0003).
type Store interface {
	// Snapshot returns the full applied configuration at the current revision.
	Snapshot(ctx context.Context) (*resources.Snapshot, error)
	// Revision returns the current monotonic revision.
	Revision(ctx context.Context) (int64, error)
	// Apply upserts resources atomically, bumping revision. dryRun validates only.
	Apply(ctx context.Context, actor string, docs []byte, dryRun bool) (revision int64, err error)
	// Watch blocks until revision > since (or ctx done), then returns latest revision.
	Watch(ctx context.Context, since int64) (int64, error)
	// Audit returns recent audit entries.
	Audit(ctx context.Context, limit int) ([]AuditEntry, error)
}

// Memory is an in-process Store for tests and all-in-one mode.
type Memory struct {
	mu       sync.RWMutex
	revision int64
	snap     *resources.Snapshot
	audit    []AuditEntry
	waiters  []chan int64
}

func NewMemory(initial *resources.Snapshot) *Memory {
	if initial == nil {
		initial = &resources.Snapshot{
			Revision:  0,
			Tenants:   map[string]*resources.Tenant{},
			Trunks:    map[string]*resources.Trunk{},
			Secrets:   map[string]string{},
			Endpoints: nil,
			Routes:    nil,
		}
	}
	return &Memory{revision: initial.Revision, snap: cloneSnap(initial)}
}

func (m *Memory) Snapshot(ctx context.Context) (*resources.Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneSnap(m.snap), nil
}

func (m *Memory) Revision(ctx context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.revision, nil
}

func (m *Memory) Apply(ctx context.Context, actor string, docs []byte, dryRun bool) (int64, error) {
	next, err := resources.ParseYAML(docs)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := ValidateSnapshot(next); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if dryRun {
		m.mu.RLock()
		rev := m.revision
		m.mu.RUnlock()
		return rev, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.revision++
	next.Revision = m.revision
	// Merge secrets from previous if not re-specified
	if m.snap != nil {
		for k, v := range m.snap.Secrets {
			if _, ok := next.Secrets[k]; !ok {
				next.Secrets[k] = v
			}
		}
	}
	m.snap = next
	m.audit = append(m.audit, AuditEntry{
		Revision: m.revision,
		Action:   "apply",
		Actor:    actor,
		At:       time.Now().UTC(),
	})
	rev := m.revision
	for _, ch := range m.waiters {
		select {
		case ch <- rev:
		default:
		}
	}
	m.waiters = nil
	return rev, nil
}

func (m *Memory) Watch(ctx context.Context, since int64) (int64, error) {
	m.mu.Lock()
	if m.revision > since {
		rev := m.revision
		m.mu.Unlock()
		return rev, nil
	}
	ch := make(chan int64, 1)
	m.waiters = append(m.waiters, ch)
	m.mu.Unlock()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case rev := <-ch:
		return rev, nil
	}
}

func (m *Memory) Audit(ctx context.Context, limit int) ([]AuditEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > len(m.audit) {
		limit = len(m.audit)
	}
	out := make([]AuditEntry, limit)
	copy(out, m.audit[len(m.audit)-limit:])
	return out, nil
}

// ValidateSnapshot performs basic structural validation.
func ValidateSnapshot(s *resources.Snapshot) error {
	if s == nil {
		return errors.New("nil snapshot")
	}
	for _, r := range s.Routes {
		if r.Spec.Action.Type == "" {
			return fmt.Errorf("route %s: missing action.type", r.Metadata.Name)
		}
		switch r.Spec.Action.Type {
		case "proxy", "loadBalance", "registerLookup", "reject", "webhook":
		default:
			return fmt.Errorf("route %s: unknown action.type %q", r.Metadata.Name, r.Spec.Action.Type)
		}
	}
	return nil
}

func cloneSnap(s *resources.Snapshot) *resources.Snapshot {
	if s == nil {
		return nil
	}
	b, _ := json.Marshal(s)
	var out resources.Snapshot
	_ = json.Unmarshal(b, &out)
	if out.Tenants == nil {
		out.Tenants = map[string]*resources.Tenant{}
	}
	if out.Trunks == nil {
		out.Trunks = map[string]*resources.Trunk{}
	}
	if out.Secrets == nil {
		out.Secrets = map[string]string{}
	}
	return &out
}
