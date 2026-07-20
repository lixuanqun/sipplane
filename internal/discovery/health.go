package discovery

import (
	"context"
	"log/slog"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// HealthChecker sends OPTIONS pings to trunks.
type HealthChecker struct {
	Client   *sipgo.Client
	Interval time.Duration
	Timeout  time.Duration
	Groups   []*DispatchGroup
	Log      *slog.Logger
}

func (h *HealthChecker) Run(ctx context.Context) {
	if h.Interval == 0 {
		h.Interval = 30 * time.Second
	}
	if h.Timeout == 0 {
		h.Timeout = 5 * time.Second
	}
	if h.Log == nil {
		h.Log = slog.Default()
	}
	t := time.NewTicker(h.Interval)
	defer t.Stop()
	h.ProbeOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.ProbeOnce(ctx)
		}
	}
}

// ProbeOnce runs a single OPTIONS pass against all group members.
func (h *HealthChecker) ProbeOnce(ctx context.Context) {
	h.probeAll(ctx)
}

func (h *HealthChecker) probeAll(ctx context.Context) {
	for _, g := range h.Groups {
		for _, m := range g.Members {
			tr := g.Trunks[m.Name]
			if tr == nil {
				continue
			}
			ok := h.ping(ctx, tr.Spec.Destination.HostPort())
			g.Mark(m.Name, ok, 5)
			if !ok {
				h.Log.Debug("trunk probe failed", "trunk", m.Name, "dest", tr.Spec.Destination.HostPort())
			}
		}
	}
}

func (h *HealthChecker) ping(ctx context.Context, dest string) bool {
	cctx, cancel := context.WithTimeout(ctx, h.Timeout)
	defer cancel()
	params := sip.NewParams()
	params.Add("tag", "health")
	req := sip.NewRequest(sip.OPTIONS, sip.Uri{Host: "sipplane.health"})
	req.AppendHeader(&sip.FromHeader{Address: sip.Uri{User: "sipplane", Host: "health"}, Params: params})
	req.AppendHeader(&sip.ToHeader{Address: sip.Uri{Host: "sipplane.health"}})
	req.SetDestination(dest)
	res, err := h.Client.Do(cctx, req)
	if err != nil {
		return false
	}
	return res.StatusCode >= 200 && res.StatusCode < 300
}
