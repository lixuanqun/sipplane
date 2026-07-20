package policy

import (
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/sipplane/sipplane/internal/config"
)

// FromConfig builds an ingress Chain from bootstrap policies (nil if unset).
// Without Redis, rate-limit is process-local.
func FromConfig(cfg config.Config) *Chain {
	return Build(cfg, nil)
}

// Build builds an ingress Chain; when rdb is non-nil and rateLimit.backend is
// "redis" or "auto"/empty with redis available, uses shared Redis limiter.
func Build(cfg config.Config, rdb redis.Cmdable) *Chain {
	if !cfg.HasPolicies() {
		return nil
	}
	ch := &Chain{}
	if a := cfg.Policies.ACL; a != nil {
		if len(a.AllowCIDRs) > 0 || len(a.DenyCIDRs) > 0 || len(a.Methods) > 0 {
			ch.ACL = &ACL{
				AllowCIDRs: append([]string(nil), a.AllowCIDRs...),
				DenyCIDRs:  append([]string(nil), a.DenyCIDRs...),
				Methods:    append([]string(nil), a.Methods...),
			}
		}
	}
	if r := cfg.Policies.RateLimit; r != nil && r.CPS > 0 {
		ch.KeyMode = strings.ToLower(strings.TrimSpace(r.Key))
		if ch.KeyMode == "" {
			ch.KeyMode = "global"
		}
		backend := strings.ToLower(strings.TrimSpace(r.Backend))
		useRedis := rdb != nil && (backend == "redis" || backend == "" || backend == "auto")
		if backend == "local" {
			useRedis = false
		}
		if useRedis {
			ch.Limiter = &RedisRateLimit{Client: rdb, CPS: r.CPS, Burst: r.Burst}
			ch.Backend = "redis"
		} else {
			ch.Limiter = &RateLimit{CPS: r.CPS, Burst: r.Burst}
			ch.Backend = "local"
		}
	}
	if ch.ACL == nil && ch.Limiter == nil {
		return nil
	}
	return ch
}
