package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/sipplane/sipplane/internal/controlplane/store"
	"github.com/sipplane/sipplane/internal/metrics"
)

// Server exposes the control-plane REST API.
type Server struct {
	Store     store.Store
	Actor     string
	AuthToken string // optional; when set, /v1/* requires Bearer token
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/snapshot", s.getSnapshot)
	mux.HandleFunc("GET /v1/revision", s.getRevision)
	mux.HandleFunc("GET /v1/watch", s.watch)
	mux.HandleFunc("POST /v1/apply", s.apply)
	mux.HandleFunc("POST /v1/dry-run", s.dryRun)
	mux.HandleFunc("GET /v1/audit", s.audit)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return BearerAuth(s.AuthToken, mux)
}

func (s *Server) getSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := s.Store.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, snap)
}

func (s *Server) getRevision(w http.ResponseWriter, r *http.Request) {
	rev, err := s.Store.Revision(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]int64{"revision": rev})
}

func (s *Server) watch(w http.ResponseWriter, r *http.Request) {
	since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	timeout := 30 * time.Second
	if t := r.URL.Query().Get("timeout"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	rev, err := s.Store.Watch(ctx, since)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			w.WriteHeader(http.StatusOK)
			writeJSON(w, map[string]any{"revision": since, "timeout": true})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]int64{"revision": rev})
}

func (s *Server) apply(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	actor := s.Actor
	if a := r.Header.Get("X-Actor"); a != "" {
		actor = a
	}
	if actor == "" {
		actor = "api"
	}
	rev, err := s.Store.Apply(r.Context(), actor, body, false)
	if err != nil {
		metrics.ConfigApplyTotal.WithLabelValues("error").Inc()
		status := http.StatusBadRequest
		if !errors.Is(err, store.ErrValidation) {
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}
	metrics.ConfigApplyTotal.WithLabelValues("ok").Inc()
	metrics.ConfigRevision.Set(float64(rev))
	writeJSON(w, map[string]int64{"revision": rev})
}

func (s *Server) dryRun(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rev, err := s.Store.Apply(r.Context(), "dry-run", body, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "revision": rev})
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}
	entries, err := s.Store.Audit(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, entries)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
