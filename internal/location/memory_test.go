package location

import (
	"errors"
	"testing"
	"time"
)

func TestMemoryStorePutGetDelete(t *testing.T) {
	s := NewMemoryStore()
	aor := "sip:alice@acme.example"
	contacts := []Contact{{
		URI:       "sip:alice@10.0.0.1:5060",
		HostPort:  "10.0.0.1:5060",
		Transport: "udp",
	}}
	if err := s.Put(aor, contacts, 60*time.Second); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(aor)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].HostPort != "10.0.0.1:5060" {
		t.Fatalf("unexpected contacts: %+v", got)
	}
	if s.Count() != 1 {
		t.Fatalf("count=%d", s.Count())
	}
	if err := s.Delete(aor); err != nil {
		t.Fatal(err)
	}
	_, err = s.Get(aor)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreExpiry(t *testing.T) {
	s := NewMemoryStore()
	aor := "sip:bob@acme.example"
	_ = s.Put(aor, []Contact{{URI: "sip:bob@1.2.3.4:5060", HostPort: "1.2.3.4:5060"}}, 20*time.Millisecond)
	time.Sleep(40 * time.Millisecond)
	_, err := s.Get(aor)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want expired ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreUnregister(t *testing.T) {
	s := NewMemoryStore()
	aor := "sip:carol@acme.example"
	_ = s.Put(aor, []Contact{{URI: "sip:carol@1.1.1.1:5060", HostPort: "1.1.1.1:5060"}}, time.Hour)
	_ = s.Put(aor, nil, 0)
	_, err := s.Get(aor)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound after unregister, got %v", err)
	}
}
