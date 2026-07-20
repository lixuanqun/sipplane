package location

import (
	"errors"
	"time"
)

// ErrNotFound is returned when no contacts are bound to an AOR.
var ErrNotFound = errors.New("location: aor not found")

// Contact is a registered SIP contact binding.
type Contact struct {
	URI       string
	HostPort  string // host:port for dialing
	Transport string
	ExpiresAt time.Time
	Raw       string // original Contact header value when available
	FlowToken string // RFC 5626 flow token when outbound is used
}

// Store is the location service interface (RFC 0005).
// P1: memory; P3: Redis (+ local cache).
type Store interface {
	Put(aor string, contacts []Contact, expires time.Duration) error
	Get(aor string) ([]Contact, error)
	Delete(aor string) error
	Count() int
}
