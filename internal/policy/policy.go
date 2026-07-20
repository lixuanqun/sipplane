package policy

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/metrics"
)

// Decision from a policy evaluation.
type Decision int

const (
	Continue Decision = iota
	Deny
)

// Result carries deny details.
type Result struct {
	Decision Decision
	Code     int
	Reason   string
}

// Limiter decides whether a request key is allowed (token bucket / shared).
type Limiter interface {
	Allow(ctx context.Context, key string) bool
}

// ACL denies or allows by source IP / method.
type ACL struct {
	AllowCIDRs []string `yaml:"allowCidrs" json:"allowCidrs"`
	DenyCIDRs  []string `yaml:"denyCidrs" json:"denyCidrs"`
	Methods    []string `yaml:"methods" json:"methods"` // if set, only these methods allowed

	allow []*net.IPNet
	deny  []*net.IPNet
	once  sync.Once
}

func (a *ACL) compile() {
	a.once.Do(func() {
		for _, c := range a.AllowCIDRs {
			if _, n, err := net.ParseCIDR(c); err == nil {
				a.allow = append(a.allow, n)
			}
		}
		for _, c := range a.DenyCIDRs {
			if _, n, err := net.ParseCIDR(c); err == nil {
				a.deny = append(a.deny, n)
			}
		}
	})
}

func (a *ACL) Check(req *sip.Request) Result {
	a.compile()
	host, _, err := net.SplitHostPort(req.Source())
	if err != nil {
		host = req.Source()
	}
	ip := net.ParseIP(host)
	if ip != nil {
		for _, n := range a.deny {
			if n.Contains(ip) {
				return Result{Decision: Deny, Code: 403, Reason: "ACL deny"}
			}
		}
		if len(a.allow) > 0 {
			ok := false
			for _, n := range a.allow {
				if n.Contains(ip) {
					ok = true
					break
				}
			}
			if !ok {
				return Result{Decision: Deny, Code: 403, Reason: "ACL not allowed"}
			}
		}
	}
	if len(a.Methods) > 0 {
		ok := false
		m := req.Method.String()
		for _, x := range a.Methods {
			if strings.EqualFold(x, m) {
				ok = true
				break
			}
		}
		if !ok {
			return Result{Decision: Deny, Code: 405, Reason: "Method Not Allowed"}
		}
	}
	return Result{Decision: Continue}
}

// RateLimit is a process-local token bucket (implements Limiter; key ignored).
type RateLimit struct {
	CPS   float64 `yaml:"cps" json:"cps"`
	Burst int     `yaml:"burst" json:"burst"`

	mu     sync.Mutex
	tokens float64
	last   time.Time
}

func (r *RateLimit) Allow(_ context.Context, _ string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if r.last.IsZero() {
		r.last = now
		r.tokens = float64(r.burst())
	}
	elapsed := now.Sub(r.last).Seconds()
	r.last = now
	r.tokens += elapsed * r.CPS
	max := float64(r.burst())
	if r.tokens > max {
		r.tokens = max
	}
	if r.tokens < 1 {
		return false
	}
	r.tokens--
	return true
}

func (r *RateLimit) burst() int {
	if r.Burst > 0 {
		return r.Burst
	}
	if r.CPS < 1 {
		return 1
	}
	return int(r.CPS)
}

// Chain runs ingress policies in order.
type Chain struct {
	ACL     *ACL
	Limiter Limiter
	// KeyMode: "global" (default) or "ip" (per source IP).
	KeyMode  string
	Backend  string // "local" | "redis" (metrics label)
}

func (c *Chain) Ingress(req *sip.Request) Result {
	if c == nil {
		return Result{Decision: Continue}
	}
	if c.ACL != nil {
		if res := c.ACL.Check(req); res.Decision == Deny {
			return res
		}
	}
	if c.Limiter != nil {
		key := "global"
		if strings.EqualFold(c.KeyMode, "ip") {
			key = sourceHost(req)
		}
		if !c.Limiter.Allow(context.Background(), key) {
			backend := c.Backend
			if backend == "" {
				backend = "local"
			}
			metrics.RateLimitRejected.WithLabelValues(backend, keyLabel(key)).Inc()
			return Result{Decision: Deny, Code: 503, Reason: "Rate Limited"}
		}
	}
	return Result{Decision: Continue}
}

func sourceHost(req *sip.Request) string {
	host, _, err := net.SplitHostPort(req.Source())
	if err != nil {
		return req.Source()
	}
	return host
}

func keyLabel(key string) string {
	if key == "" || key == "global" {
		return "global"
	}
	return "ip"
}
