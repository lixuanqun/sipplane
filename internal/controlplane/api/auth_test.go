package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipplane/sipplane/internal/controlplane/api"
	"github.com/sipplane/sipplane/internal/controlplane/store"
)

func TestBearerAuth(t *testing.T) {
	mem := store.NewMemory(nil)
	srv := &api.Server{Store: mem, Actor: "test", AuthToken: "secret"}
	h := srv.Handler()

	// healthz anonymous
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("healthz=%d", rec.Code)
	}

	// missing token
	req = httptest.NewRequest(http.MethodGet, "/v1/revision", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}

	// wrong token
	req = httptest.NewRequest(http.MethodGet, "/v1/revision", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}

	// good token
	req = httptest.NewRequest(http.MethodGet, "/v1/revision", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("want 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBearerAuthDisabled(t *testing.T) {
	mem := store.NewMemory(nil)
	srv := &api.Server{Store: mem, Actor: "test"}
	h := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/v1/revision", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("empty token should allow, got %d", rec.Code)
	}
}
