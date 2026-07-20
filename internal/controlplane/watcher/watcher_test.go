package watcher_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sipplane/sipplane/internal/controlplane/watcher"
	"github.com/sipplane/sipplane/internal/resources"
)

type fakeApplier struct {
	rev   atomic.Int64
	ready atomic.Bool
}

func (f *fakeApplier) ReplaceSnapshot(snap *resources.Snapshot) {
	f.rev.Store(snap.Revision)
}
func (f *fakeApplier) SetReady(v bool) { f.ready.Store(v) }

func TestWatcherFetchAndApply(t *testing.T) {
	var rev atomic.Int64
	rev.Store(1)
	var sawAuth atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer test-token" {
			sawAuth.Store(true)
		}
		switch r.URL.Path {
		case "/v1/snapshot":
			_ = json.NewEncoder(w).Encode(&resources.Snapshot{
				Revision: rev.Load(),
				Tenants:  map[string]*resources.Tenant{},
				Trunks:   map[string]*resources.Trunk{},
				Secrets:  map[string]string{},
			})
		case "/v1/watch":
			_ = json.NewEncoder(w).Encode(map[string]any{"revision": rev.Load(), "timeout": true})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	app := &fakeApplier{}
	w := &watcher.Watcher{
		BaseURL:     srv.URL,
		Token:       "test-token",
		Applier:     app,
		StaleAfter:  time.Minute,
		PollTimeout: 50 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Run(ctx) }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if app.rev.Load() == 1 && app.ready.Load() {
			if !sawAuth.Load() {
				t.Fatal("expected Authorization header")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("watcher did not apply snapshot, rev=%d ready=%v", app.rev.Load(), app.ready.Load())
}
