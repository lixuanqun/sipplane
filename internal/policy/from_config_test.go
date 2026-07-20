package policy_test

import (
	"testing"

	"github.com/sipplane/sipplane/internal/config"
	"github.com/sipplane/sipplane/internal/policy"
)

func TestFromConfigNilWhenEmpty(t *testing.T) {
	if policy.FromConfig(config.Config{}) != nil {
		t.Fatal("expected nil chain")
	}
}

func TestFromConfigACLAndRate(t *testing.T) {
	cfg := config.Config{
		Policies: config.PoliciesConfig{
			ACL: &config.ACLConfig{
				DenyCIDRs: []string{"10.0.0.0/8"},
			},
			RateLimit: &config.RateLimitConfig{CPS: 10, Burst: 5},
		},
	}
	ch := policy.FromConfig(cfg)
	if ch == nil || ch.ACL == nil || ch.Limiter == nil {
		t.Fatalf("%+v", ch)
	}
	rl, ok := ch.Limiter.(*policy.RateLimit)
	if !ok || rl.CPS != 10 || rl.Burst != 5 {
		t.Fatalf("%+v", ch.Limiter)
	}
	if ch.Backend != "local" {
		t.Fatalf("backend=%s", ch.Backend)
	}
}

func TestBuildForcesLocal(t *testing.T) {
	cfg := config.Config{
		Policies: config.PoliciesConfig{
			RateLimit: &config.RateLimitConfig{CPS: 5, Burst: 2, Backend: "local"},
		},
	}
	ch := policy.Build(cfg, nil)
	if _, ok := ch.Limiter.(*policy.RateLimit); !ok {
		t.Fatalf("want local limiter, got %T", ch.Limiter)
	}
}
