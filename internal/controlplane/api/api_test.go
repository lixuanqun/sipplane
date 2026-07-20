package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sipplane/sipplane/internal/controlplane/api"
	"github.com/sipplane/sipplane/internal/controlplane/store"
)

func TestApplyDryRunAndWatch(t *testing.T) {
	mem := store.NewMemory(nil)
	srv := &api.Server{Store: mem, Actor: "test"}
	h := srv.Handler()

	yaml := []byte(`
apiVersion: sipplane.io/v1alpha1
kind: Route
metadata:
  name: r1
spec:
  priority: 1
  match:
    methods: ["INVITE"]
  action:
    type: registerLookup
`)

	// dry-run
	req := httptest.NewRequest(http.MethodPost, "/v1/dry-run", bytes.NewReader(yaml))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("dry-run status=%d body=%s", rec.Code, rec.Body.String())
	}

	// apply
	req = httptest.NewRequest(http.MethodPost, "/v1/apply", bytes.NewReader(yaml))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("apply status=%d body=%s", rec.Code, rec.Body.String())
	}
	var applied map[string]int64
	_ = json.Unmarshal(rec.Body.Bytes(), &applied)
	if applied["revision"] != 1 {
		t.Fatalf("revision=%v", applied)
	}

	// snapshot
	req = httptest.NewRequest(http.MethodGet, "/v1/snapshot", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}

	// watch should return immediately when since < current
	req = httptest.NewRequest(http.MethodGet, "/v1/watch?since=0&timeout=1s", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}

	// invalid apply rejected without bump
	bad := []byte(`kind: Route
metadata: {name: x}
spec:
  action: {type: bogus}
`)
	req = httptest.NewRequest(http.MethodPost, "/v1/apply", bytes.NewReader(bad))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code < 400 {
		t.Fatalf("expected error, got %d", rec.Code)
	}
	rev, _ := mem.Revision(context.Background())
	if rev != 1 {
		t.Fatalf("revision bumped on bad apply: %d", rev)
	}

	_ = time.Second
}
