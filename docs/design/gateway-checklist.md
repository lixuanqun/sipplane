# Gateway-patterns checklist (pre-GA)

Track implementation against [gateway-patterns.md](gateway-patterns.md).  
Legend: `Done` / `Partial` / `Open`.

| Pattern area | Expectation | Status | Evidence |
|--------------|-------------|--------|----------|
| Declarative resources | Tenant/Endpoint/Trunk/Route YAML + apply | Done | resources + control plane |
| Validate / dry-run | Invalid apply does not bump revision | Done | store/api tests |
| Revisioned config | Monotonic revision; DP watch | Done | RFC 0002 + watcher |
| Last-known-good | CP outage keeps snapshot | Done | watcher behavior |
| Stale → not ready | `/readyz` 503 | Done | watcher StaleAfter |
| Ingress policy chain | ACL + rate limit | Done | docs/policies.md |
| Observability | Prometheus + access log + healthz/readyz | Done | dataplane |
| Tracing | OTel optional | Partial | otelx basic |
| HEP | Optional capture | Done | docs/edge.md |
| Upstream discovery | Trunk + loadBalance algorithms | Done | docs/cluster.md |
| Active health | OPTIONS ping + eject | Done | discovery health |
| Affinity | Call-ID hash (LB + route) | Done | RFC 0001 + cluster.md |
| Fail-closed location | 480/503 | Done | RFC 0005 |
| advertised_host | Required non-loopback | Done | RFC 0004 |
| GitOps-friendly ctl | sipplanectl apply | Done | control-plane.md |
| Webhook extension | Route webhook + timeout fallback | Done | edge.md |
| Wasm / rich plugins | Optional later | Open | ROADMAP P4 |
| Dashboard | Optional | Open | ROADMAP P4 |
| Shared rate-limit | Multi-instance Redis bucket | Done | docs/policies.md `backend: redis` |
| CP authn/z | Bearer token (`/v1/*`) | Partial | control-plane.md; mTLS/RBAC open |
| NetworkPolicy / PSS | Helm non-root + NP | Done | deploy-production.md |

**Pre-GA goal:** all rows except explicitly optional (Wasm/Dashboard) are `Done` or accepted `Partial` with linked residual Issues.
