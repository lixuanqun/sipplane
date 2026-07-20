# Gateway patterns for sipplane

> Status: **Draft**. Canonical English. 中文：[gateway-patterns.zh-CN.md](gateway-patterns.zh-CN.md)

sipplane is a **SIP signaling plane**, not an HTTP API gateway.
We deliberately learn product patterns from mature open-source gateways
(APISIX, Kong, Traefik, Tyk, KrakenD, Easegress, Envoy/Istio, Caddy)
and map them onto SIP semantics (transactions, dialogs, REGISTER, trunks).

**Principle:** borrow **ops model and control-plane UX**; keep **SIP protocol correctness**.

## 1. What we borrow (summary)

| Capability | Reference gateways | sipplane mapping |
|------------|-------------------|------------------|
| Policy / plugin chain | APISIX, Kong, Easegress, Envoy | Ordered SIP filter phases |
| Observability | APISIX, Traefik, Envoy | Metrics + structured access log + HEP |
| Control / data split | APISIX (etcd), Envoy (xDS), Caddy Admin API | Admin API + revision Watch |
| Service discovery | Traefik, Istio, go-zero | Backend / trunk discovery + health |
| Upstream health & LB | APISIX Upstream, Kong | `DispatchGroup` + OPTIONS / outlier |
| Consumer / identity | Kong, Tyk | `Endpoint` + `Trunk` credentials |
| Rate limit / quotas | APISIX, Tyk | Tenant / trunk / IP / endpoint limits |
| Declarative GitOps | KrakenD, Traefik CRDs | `sipplanectl apply` + same API |
| Hot reload | APISIX, Caddy | Atomic snapshot swap, no SIP bounce |

## 2. Policy engine (strategies)

### 2.1 Filter chain (Easegress / Envoy / APISIX)

HTTP gateways run middleware/filters in phases. sipplane adopts the same idea for **SIP messages**:

```text
ingress → auth → routing → egress → async
```

| Phase | Purpose | Example policies |
|-------|---------|------------------|
| **ingress** | Admit / shape traffic early | IP ACL, method ACL, CPS limit, header normalize |
| **auth** | Prove identity | Digest, IP trust for trunks, mTLS (TLS SIP) |
| **routing** | Decide where to send | Route match, number rewrite, LB, register lookup |
| **egress** | Prepare outbound hop | Trunk auth, Record-Route, topology hiding |
| **async** | Side effects | CDR, webhook, event bus, HEP mirror |

Rules:

1. Policies are **resources** (or bindings on Route / Tenant / Trunk), not hard-coded `if` trees.
2. Each policy has **priority**, enable/disable, and typed config (JSON/YAML schema).
3. Failure modes are explicit: `deny` | `continue` | `fallback` (with timeout).
4. Hot-path policies must be **local** (cached). External webhook/gRPC policy = hard timeout + default action (APISIX external-auth pattern).

### 2.2 First-party policies (roadmap)

| Policy | Analog | Target version |
|--------|--------|----------------|
| `acl` | APISIX ip-restriction | v0.2 |
| `rate_limit` | limit-req / limit-count | v0.2–v0.3 |
| `digest_auth` | key-auth / basic-auth | v0.1 |
| `metrics` | prometheus | v0.1 |
| `access_log` | access-log | v0.1 |
| `number_transform` | proxy-rewrite | v0.2+ |
| `circuit_breaker` | APISIX / Easegress | v0.3 |
| `webhook_policy` | serverless / forward-auth | v0.4 |
| `wasm` / `grpc_plugin` | Wasm / external plugin | v0.4+ |

### 2.3 What we do **not** copy from HTTP gateways

- Path/Host-only matching as the primary model → SIP uses Request-URI, To/From, trunk source, PAI, etc.
- Stateless request/response only → SIP needs dialog/transaction awareness.
- Body-centric transforms as default → SDP changes imply media-plane implications.

## 3. Observability

Gateways win operators by making every request **measurable and explainable**.

### 3.1 Metrics (Prometheus) — required from v0.1

Labeled like APISIX/Traefik (`route_id`, `upstream`, `status`):

```text
sipplane_sip_requests_total{method,tenant,route,trunk,code}
sipplane_sip_transactions_inflight{method}
sipplane_register_bindings{tenant}
sipplane_upstream_health{trunk,state}          # 0/1
sipplane_config_revision                       # gauge
sipplane_config_apply_total{result}
sipplane_policy_latency_seconds{policy,phase}
sipplane_rate_limit_rejected_total{dimension}
```

### 3.2 Structured access log

One log line (or JSON event) per SIP transaction completion, including:

- `call_id`, `method`, `from_tag`, `to_tag` (when present)
- `tenant`, `route`, `trunk`, `config_revision`
- `src`, `dst`, `response_code`, `duration_ms`
- `policy_trace` (optional: which policies ran)

### 3.3 Tracing & HEP

| Signal | Source pattern | sipplane |
|--------|----------------|----------|
| OpenTelemetry | Envoy / Traefik | Control-plane RPCs first; SIP spans later (sampled) |
| HEP → Homer | Kamailio / OpenSIPS / HEP agents | v0.4 — SIP-native “pcap access log” |
| Health / ready | K8s probes | `/healthz` always; `/readyz` fails if config stale or critical upstreams down |

### 3.4 Debuggability

- Admin API: dump **current revision**, route table hash, trunk health.
- Optional `sipplane debug dump --call-id` (future) — analogous to gateway request-id lookup.

## 4. Control plane / data plane separation

### 4.1 Roles (APISIX + Envoy + Caddy)

```text
┌──────────────────────────┐         Watch / snapshot
│  Control plane           │ ──────────────────────────► Data plane pods
│  Admin API · validate    │         revision N
│  store · audit · GitOps  │
└──────────────────────────┘
         ▲
         │ sipplanectl / CI / Dashboard
```

| Concern | Control plane | Data plane |
|---------|---------------|------------|
| CRUD resources | Yes | No |
| SIP I/O | No | Yes |
| Authoritative config | Yes (store) | Cached snapshot only |
| Location / dialog state | No (except tooling) | State plane (Redis) |
| Survive CP outage | — | **Yes** — last-known-good |

### 4.2 Config distribution semantics (learn from xDS / APISIX / Caddy)

| Rule | Default for sipplane |
|------|----------------------|
| Delivery | Full snapshot + monotonic `revision` (v0.2–v0.3); incremental later |
| Apply | Atomic swap; failed validate → keep previous revision |
| CP down | Data plane keeps serving; emit alert; optional `ready=false` if stale > SLA |
| Validate | `POST /validate` / dry-run before commit (APISIX schema check) |
| Audit | Who changed what, old/new revision |

### 4.3 Admin surface

Inspired by APISIX Admin API + Caddy Admin API + `kubectl`/`apisixctl`:

- REST and/or gRPC management API
- `sipplanectl apply -f ./manifests/` (GitOps-friendly)
- OpenAPI for humans and automation
- Dashboard: **after** API stability (do not block v0.2 on UI)

## 5. Automatic service discovery

Traefik/Istio discover backends from Docker/K8s/Consul.
sipplane discovers **SIP backends** (trunks, media edges, app servers).

### 5.1 Discovery sources (phased)

| Source | Use | Phase |
|--------|-----|-------|
| Static YAML / API | Dev & small prod | v0.1–v0.2 |
| Control-plane `Trunk` / `DispatchGroup` | Primary | v0.2+ |
| DNS SRV / A refresh | Carrier / remote SBC | v0.3+ |
| Kubernetes Endpoints / EndpointSlice | In-cluster FreeSWITCH / LiveKit SIP | v0.4+ |
| Consul / Nacos (optional) | Enterprise registries | later |

### 5.2 Health & load balancing (APISIX Upstream pattern)

Discovery alone is insufficient — gateways continuously **probe and eject**:

```yaml
kind: DispatchGroup
metadata:
  name: media-farm
spec:
  algorithm: consistent_hash   # round_robin | weighted | least_sessions
  hashKey: call-id             # SIP-critical (HTTP gateways rarely need this)
  members:
    - ref: trunk-fs-a
      weight: 100
  healthCheck:
    active:
      method: OPTIONS
      interval: 30s
      timeout: 5s
      healthyThreshold: 2
      unhealthyThreshold: 3
    passive:
      consecutiveFailures: 5
      ejectSeconds: 30
```

Behaviors to copy from HTTP gateways:

- Active + passive health
- Outlier detection / temporary eject
- All-unhealthy → deterministic **503** (no blackhole)
- Metrics per member

### 5.3 SIP-specific discovery rules

1. **REGISTER location** is itself a discovery system (AOR → Contact); treat it as first-class, not a side table.
2. **Call-ID affinity** (or shared dialog state) is mandatory for multi-instance — Traefik sticky cookie is the wrong tool; use consistent hash or Redis dialog.
3. UDP peers have no TCP connection stickiness — prefer shared state over L4 sticky alone.

## 6. Capability matrix vs reference gateways

| Feature | APISIX | Traefik | Tyk | Easegress | KrakenD | sipplane target |
|---------|--------|---------|-----|-----------|---------|-----------------|
| Hot config | ✓ etcd | ✓ providers | ✓ | ✓ | file reload | ✓ Watch + revision |
| Plugin chain | ✓ | middleware | ✓ | pipeline | flexible | ✓ SIP phases |
| Service discovery | ✓ | ✓ strong | ✓ | ✓ | static/file | ✓ trunks + K8s later |
| Upstream health | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ OPTIONS + outlier |
| Multi-tenant | ✓ | weak | ✓ | ✓ | weak | ✓ Tenant |
| Full API portal | commercial | — | ✓ | — | — | later / optional |
| SIP registrar | — | — | — | — | — | **native** |
| HEP / Homer | — | — | — | — | — | **native** |

## 7. Non-goals (do not become HTTP gateway)

- Replace Traefik/APISIX for HTTP/gRPC north-south traffic
- GraphQL aggregation / response composition (KrakenD specialty)
- Full developer portal on day one
- Drop-in Kamailio `.cfg` compatibility

## 8. Design checklist for contributors

Before adding a feature, ask:

1. Which gateway pattern does this map to? (policy / observe / CP-DP / discovery)
2. Is it on the SIP hot path? If yes: local cache + timeout + fail mode.
3. Is it a resource, a policy binding, or a data-plane builtin?
4. What metrics and log fields prove it works?
5. What is the SIP-specific difference from the HTTP analog?

## 9. Related docs

- [Architecture](../architecture.md)
- [Resource model](resource-model.md)
- [Design principles](principles.md)
- [Roadmap](../../ROADMAP.md)
