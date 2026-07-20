package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/sipplane/sipplane/internal/metrics"
	"github.com/sipplane/sipplane/internal/resources"
)

// Applier receives new snapshots from the control plane.
type Applier interface {
	ReplaceSnapshot(snap *resources.Snapshot)
	SetReady(v bool)
}

// Watcher polls/long-polls a control-plane URL for config revisions (RFC 0002).
type Watcher struct {
	BaseURL         string
	Token           string // optional Bearer token
	HTTP            *http.Client
	Applier         Applier
	Log             *slog.Logger
	StaleAfter      time.Duration
	PollTimeout     time.Duration
	lastSync        time.Time
	currentRevision int64
}

func (w *Watcher) Run(ctx context.Context) error {
	if w.HTTP == nil {
		w.HTTP = &http.Client{Timeout: 60 * time.Second}
	}
	if w.Log == nil {
		w.Log = slog.Default()
	}
	if w.StaleAfter == 0 {
		w.StaleAfter = 60 * time.Second
	}
	if w.PollTimeout == 0 {
		w.PollTimeout = 25 * time.Second
	}

	if err := w.fetchAndApply(ctx); err != nil {
		w.Log.Warn("initial snapshot fetch failed", "err", err)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if time.Since(w.lastSync) > w.StaleAfter {
				w.Applier.SetReady(false)
			}
			watchCtx, cancel := context.WithTimeout(ctx, w.PollTimeout+2*time.Second)
			rev, timedOut, err := w.watch(watchCtx, w.currentRevision)
			cancel()
			if err != nil {
				w.Log.Debug("watch error", "err", err)
				continue
			}
			if timedOut {
				continue
			}
			if rev > w.currentRevision {
				if err := w.fetchAndApply(ctx); err != nil {
					w.Log.Error("apply snapshot failed", "err", err)
				}
			}
		}
	}
}

func (w *Watcher) watch(ctx context.Context, since int64) (rev int64, timedOut bool, err error) {
	url := fmt.Sprintf("%s/v1/watch?since=%d&timeout=%s", w.BaseURL, since, w.PollTimeout)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, false, err
	}
	w.setAuth(req)
	res, err := w.HTTP.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		return 0, false, fmt.Errorf("watch unauthorized")
	}
	var body struct {
		Revision int64 `json:"revision"`
		Timeout  bool  `json:"timeout"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return 0, false, err
	}
	return body.Revision, body.Timeout, nil
}

func (w *Watcher) fetchAndApply(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.BaseURL+"/v1/snapshot", nil)
	if err != nil {
		return err
	}
	w.setAuth(req)
	res, err := w.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("snapshot status %d", res.StatusCode)
	}
	var snap resources.Snapshot
	if err := json.NewDecoder(res.Body).Decode(&snap); err != nil {
		return err
	}
	w.Applier.ReplaceSnapshot(&snap)
	w.currentRevision = snap.Revision
	w.lastSync = time.Now()
	w.Applier.SetReady(true)
	metrics.ConfigRevision.Set(float64(snap.Revision))
	w.Log.Info("config snapshot applied", "revision", snap.Revision)
	return nil
}

func (w *Watcher) setAuth(req *http.Request) {
	if w.Token != "" {
		req.Header.Set("Authorization", "Bearer "+w.Token)
	}
}
