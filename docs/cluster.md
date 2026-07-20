# Cluster & discovery (P3)

Goals for multi-instance sipplane ([RFC 0001](design/rfc/0001-affinity.md), [RFC 0005](design/rfc/0005-location-cache.md)):

1. **Shared REGISTER location** (Redis) so any data-plane pod can route to a Contact.
2. **Call-ID affinity** so CANCEL / in-dialog requests hit the same process when possible.
3. **Upstream health** via OPTIONS + outlier eject for trunks.

## Topology

```text
UA / Trunk
    │
    ▼
 External LB (optional)  ── hash(Call-ID) ──► sipplane-A / sipplane-B
    │                                              │
    │                                              ▼
    │                                         Redis location
    │
    └── or: single VIP → one sipplane; affinity inside Route loadBalance
```

Two complementary layers:

| Layer | Mechanism | When |
|-------|-----------|------|
| **Front LB** | HAProxy / Envoy hash on Call-ID | Multiple sipplane pods |
| **Route `loadBalance`** | `algorithm: consistent_hash` on trunks | Multiple media trunks behind one sipplane |

## Redis location

```bash
# bootstrap / env
redis_addr: "127.0.0.1:6379"
# SIPPLANE_REDIS_ADDR=127.0.0.1:6379
```

| Detail | Value |
|--------|--------|
| Key prefix | `sipplane:loc:` + AOR |
| Local cache TTL | ~5s |
| Lookup timeout | ~100ms |
| Miss / error | fail-closed → 480 / 503 |

**Multi-tenant isolation:** keys are AOR strings today (include tenant in AOR host, e.g. `sip:alice@acme.example`). Dedicated Redis hash-tags per tenant are still open.

Test: `go test ./internal/location -run Redis -count=1` (needs Redis).

## Route loadBalance

See [examples/config/lab-lb.yaml](../examples/config/lab-lb.yaml).

```yaml
kind: Route
metadata:
  name: to-media-farm
spec:
  match:
    methods: ["INVITE"]
  action:
    type: loadBalance
    algorithm: consistent_hash   # round_robin | weighted | consistent_hash
    trunks:
      - name: fs-a
        weight: 100
      - name: fs-b
        weight: 100
```

| Algorithm | Behavior |
|-----------|----------|
| `consistent_hash` / `call-id` | Hash Call-ID → stable member among healthy |
| `weighted` | Highest weight among healthy (default if unset) |
| `round_robin` | Rotate among healthy |

Implementation: `internal/discovery.DispatchGroup` via `internal/routing.Engine`.

## OPTIONS health (Trunk)

```yaml
kind: Trunk
metadata:
  name: fs-a
spec:
  destination: { host: "10.0.0.1", port: 5060 }
  options:
    sendOptionsPing: true
```

Data plane builds a ping group at startup (`GroupsFromPingTrunks`). Consecutive failures eject the member from LB picks (`Mark` / `ejectAfter=5`).

**Note:** YAML `kind: DispatchGroup` remains the **target CRD schema**; runtime today uses Trunk flags + Route `loadBalance`. Full CP loader for DispatchGroup is still open.

## Front LB example (HAProxy)

[examples/deploy/haproxy-callid.cfg](../examples/deploy/haproxy-callid.cfg) — TCP/UDP notes for Call-ID hashing. Prefer hashing at L7 SIP-aware LB when available; plain L4 UDP sticky is weak.

## Residual (not blocking P3 core close)

- Node registration / membership across DP pods  
- DNS SRV refresh for trunk destinations  
- Circuit_breaker policy resource  
- Optional full dialog store ([BACKLOG B6](design/BACKLOG.md))  
- Published load-test report  

## Shared rate limit

With `redis_addr` set, ingress `rateLimit.backend: redis` (or omit / `auto`) shares a token bucket across pods. See [policies.md](policies.md).

## Related

- Interop: [interop/README.md](interop/README.md)  
- Testing: [testing.md](testing.md)  
- RFC 0001: [design/rfc/0001-affinity.md](design/rfc/0001-affinity.md)
