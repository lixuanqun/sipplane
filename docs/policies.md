# Ingress policies (P2b)

sipplane runs an **ingress policy chain** before registrar/proxy handlers:

1. **ACL** — deny/allow by source CIDR; optional method allow-list  
2. **RateLimit** — token bucket (CPS + burst) → `503 Rate Limited`

Implementation: `internal/policy`. Wired from data-plane bootstrap via `policies:` (see below).

## Bootstrap example

```yaml
# examples/config/bootstrap.yaml
policies:
  acl:
    denyCidrs:
      - "10.255.255.0/24"     # explicit deny wins
    allowCidrs: []            # empty = allow all (that are not denied)
    # methods: ["REGISTER", "INVITE", "ACK", "BYE", "CANCEL", "OPTIONS"]
  rateLimit:
    cps: 100
    burst: 20
```

Enable by uncommenting the block in `examples/config/bootstrap.yaml`, then:

```bash
go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config
```

Logs: `ingress policies enabled`.

### ACL rules

| Setting | Effect |
|---------|--------|
| `denyCidrs` | Match → `403 ACL deny` |
| `allowCidrs` non-empty | Source must match one → else `403 ACL not allowed` |
| `allowCidrs` empty | No allow-list (only deny list applies) |
| `methods` non-empty | Other methods → `403` |

### Rate limit

| Setting | Effect |
|---------|--------|
| `cps` | Sustained requests/sec |
| `burst` | Token bucket size (default ≈ max(1, cps)) |
| `backend` | `local` \| `redis` \| `auto` (default: Redis when `redis_addr` set, else local) |
| `key` | `global` (default) or `ip` (per source IP) |

```yaml
rateLimit:
  cps: 100
  burst: 20
  backend: redis   # shared across data-plane pods
  key: ip          # optional per-source abuse control
```

| Backend | Scope | Failure |
|---------|-------|---------|
| `local` | Process-local bucket | N/A |
| `redis` | Shared (`sipplane:rl:` keys) | Redis error → **deny** (fail-closed) |

Metric: `sipplane_rate_limit_rejected_total{backend,key}`.

## Programmatic use

```go
opts := dataplane.Options{
  Policies: policy.Build(cfg, rdb), // rdb optional; nil → local limiter
  // or: &policy.Chain{ACL: &policy.ACL{...}, Limiter: &policy.RateLimit{CPS: 50}},
}
dataplane.New(cfg, snap, log, opts)
```

Unit / Redis tests: `internal/policy/*_test.go` (Redis tests skip if unavailable).

## Number transform (library)

`internal/transform` rewrites URI user parts (LCR / strip `+86`, …). **Not yet** a Route action in the proxy path — call `transform.ApplyUser` from custom glue or future Route wiring.

```go
out := transform.ApplyUser("sip:+8613800138000@acme.example", []transform.Rule{{
  Match: `^\+86(\d+)$`, Replace: "$1",
}})
```

## Related

- Control plane apply/dry-run: [control-plane.md](control-plane.md)
- Cluster / Redis: [cluster.md](cluster.md)
- Gateway patterns: [design/gateway-patterns.md](design/gateway-patterns.md)
