# Roadmap

Milestones gate on quality, not calendar pressure. **Critical defaults:** [docs/design/rfc/](docs/design/rfc/README.md) · **Deferred / promoted:** [docs/design/BACKLOG.md](docs/design/BACKLOG.md)

| Phase | Version | Theme | Status |
|-------|---------|-------|--------|
| P0 | — | Docs, governance, **critical RFCs** | **Done** |
| P1 | v0.1.0 | Callable single-node MVP | **Done** (core + examples/interop) |
| P2a | v0.2.0 | Control plane hot reload (core) | **Done** |
| P2b | v0.2.x | Policies + ctl polish | **Done** |
| P3 | v0.3.0 | Cluster state + discovery | **Done** (core + docs; residuals listed) |
| P4 | v0.4.x | Production edge hardening | **Partial** — landed features documented ([edge.md](docs/edge.md)) |
| GA | v1.0.0 | Stable API + interop matrix | **Prep** (drafts started; not tagged) |

---

## P0 — Foundation

**Status: Done** (docs accepted; remaining polish tracked below).

- [x] Public vision (README)
- [x] Architecture draft
- [x] Resource model draft
- [x] Comparison / positioning
- [x] Gateway patterns draft (policy / observe / CP-DP / discovery)
- [x] **Critical RFCs 0001–0005** (affinity, revision, store, Record-Route, location cache)
- [x] Explicit backlog document
- [x] Apache-2.0 license
- [x] Contributing & security policy
- [x] Go test CI ([`.github/workflows/test.yml`](.github/workflows/test.yml))
- [ ] Community RFC: freeze remaining `v1alpha1` resource field names (carried into implementation; non-blocking for P0 close)
- [ ] Optional: markdown docs lint / Logo / social preview

**Exit criteria:** RFCs 0001–0005 treated as defaults (**met**); P1 scope agreed (**met**); good-first-issue label for newcomers (**use** `good first issue` on GitHub).

---

## P1 — v0.1.0 “Hello call”

**Status: Done** (core runtime + lab examples + interop notes).

### Include

- [x] Go module + `cmd/sipplane` binary
- [x] UDP (+ TCP) listen via sipgo
- [x] Stateful proxy: INVITE, ACK, BYE, CANCEL, OPTIONS
- [x] **`advertised_host` required** ([RFC 0004](docs/design/rfc/0004-record-route.md)); Via / Record-Route
- [x] Registrar with **`LocationStore` interface** + in-memory impl ([RFC 0005](docs/design/rfc/0005-location-cache.md))
- [x] Digest auth (REGISTER)
- [x] Local YAML resources mapped to [resource model](docs/design/resource-model.md)
- [x] Prometheus metrics + `/healthz` `/readyz`
- [x] Structured access log (call_id, method, code, duration, …)
- [x] `examples/docker-compose` (+ test compose)
- [x] Automated CANCEL / REGISTER+INVITE tests ([docs/testing.md](docs/testing.md))
- [x] SIPp scenarios: OPTIONS + Digest REGISTER ([examples/sipp/](examples/sipp/)); INVITE/CANCEL via Go e2e
- [x] Interop notes: FreeSWITCH / Asterisk / softphones ([docs/interop/README.md](docs/interop/README.md))
- [x] SIPp OPTIONS + REGISTER smoke in CI (`scripts/sipp-smoke.sh` + workflow job)
- [x] OPTIONS / TCP Go e2e + Prometheus + access log unit tests

**Exit criteria (core):** Happy-path call + automated regression including CANCEL; `LocationStore` stable for Redis — **met**. Interop notes + SIPp REGISTER/OPTIONS — **met**.

---

## P2a — v0.2.0 Control plane (core)

**Status: Done** · Guide: [docs/control-plane.md](docs/control-plane.md)

**Goal:** Change a Route without restarting the data plane.

- [x] Management API (**REST** primary) for Tenant / Endpoint / Trunk / Route
- [x] **PostgreSQL** resource store ([RFC 0003](docs/design/rfc/0003-config-store.md)) + Memory store for lab
- [x] Watch / snapshot with monotonic `revision` ([RFC 0002](docs/design/rfc/0002-config-revision.md))
- [x] Atomic apply + validation failure does not bump revision
- [x] **Validate / dry-run** before commit
- [x] Minimal audit (who/when/revision)
- [x] `sipplane-control` + `sipplanectl`
- [x] Access log fields include tenant/route/trunk/revision where applicable
- [x] DP Watcher; stale sync can flip `/readyz`

**Exit criteria (core):** dry-run / revision / last-known-good path — **met** for lab and CI.

---

## P2b — v0.2.x Policies & tooling

**Status: Done** · Guide: [docs/policies.md](docs/policies.md)

- [x] Policy bindings: `acl` + `rate_limit` (ingress) — bootstrap `policies:` + `policy.FromConfig`
- [x] `sipplanectl apply` / dry-run / revision / snapshot
- [x] `number_transform` library (**promoted** from [BACKLOG B2](docs/design/BACKLOG.md); Route action wiring still open)
- [x] Policy cookbook + control-plane docs

**Exit criteria:** ACL + rate-limit chain + ctl apply + docs — **met**.

---

## P3 — v0.3.0 Cluster + discovery

**Status: Done (core)** · Guide: [docs/cluster.md](docs/cluster.md)

**Affinity default:** [RFC 0001](docs/design/rfc/0001-affinity.md).

- [x] Redis `LocationStore` + local cache ([RFC 0005](docs/design/rfc/0005-location-cache.md))
- [x] **DispatchGroup** algorithms wired into Route `loadBalance` (`consistent_hash` / `round_robin` / `weighted`)
- [x] **Active OPTIONS health checks** + passive outlier eject (Trunk `sendOptionsPing`)
- [x] LB / consistent-hash docs + examples (`examples/config/lab-lb.yaml`, `examples/deploy/`)
- [ ] Node registration / health across DP pods
- [ ] DNS SRV refresh for trunks (optional flag)
- [ ] Basic multi-tenant Redis key isolation (beyond AOR naming)
- [ ] `circuit_breaker` policy on trunk selection
- [ ] Optional: full dialog store ([BACKLOG B6](docs/design/BACKLOG.md))
- [ ] Full `kind: DispatchGroup` CP resource loader (target schema documented)

**Exit criteria (core):** Redis location + dispatch + OPTIONS + deployment docs — **met**. HA load report / node membership — open residuals.

---

## P4 — v0.4.x Production edge

**Guide:** [docs/edge.md](docs/edge.md) · Helm: [deploy/helm/sipplane/README.md](deploy/helm/sipplane/README.md)

| # | Item | Status |
|---|------|--------|
| 1 | SIP TLS (+ WSS later) | **TLS done** + docs; WSS open |
| 2 | NAT / Path / Outbound 5626 ([B3](docs/design/BACKLOG.md)) | **Done** + docs |
| 3 | RTPEngine control | Open |
| 4 | HEP → Homer | **Done** + docs |
| 5 | Helm chart | **Done** + values for Path/HEP/Redis/OTel |
| 6 | Kubernetes EndpointSlice discovery | Open |
| 7 | Webhook routing plugin | **Done** + docs |
| 8 | Wasm exploration | Open |
| 9 | OpenTelemetry | **Basic done** + docs |
| 10 | Dashboard | Open |
| 11 | Parallel fork ([B1](docs/design/BACKLOG.md)) | Open |
| — | 302 redirect policies ([B4](docs/design/BACKLOG.md)) | **Done** (`redirect_policy` bootstrap) |

**Landed subset exit:** operators can enable TLS/Path/Outbound/HEP/Webhook/redirect/OTel/Helm using docs — **met**. Full P4 (media control, WSS, Wasm, Dashboard) — open.

---

## v1.0.0 GA

**Prep status:** threat model + interop matrix + gateway checklist + production Helm reference exist; API freeze and external audit still open.

- [ ] `sipplane.io/v1` API (no breaking changes without major version) — still on `v1alpha1`
- [x] Interop matrix **template** ([docs/interop/matrix.md](docs/interop/matrix.md)) — fill Pass cells toward GA
- [x] Threat model **draft** ([docs/threat-model.md](docs/threat-model.md)) — formal audit still open
- [x] Gateway-patterns checklist tracker ([docs/design/gateway-checklist.md](docs/design/gateway-checklist.md)) — green when residuals closed
- [x] RFCs 0001–0005 upheld as defaults (supersede only via RFC PR)
- [x] Production **reference** deployment docs + Helm overlay ([docs/deploy-production.md](docs/deploy-production.md), [examples/deploy/production-values.yaml](examples/deploy/production-values.yaml)) — live customer reference still welcome
- [x] Control-plane Bearer auth + shared Redis rate-limit (see [control-plane.md](docs/control-plane.md), [policies.md](docs/policies.md)); mTLS/RBAC still open
- [ ] External security review notes before tag `v1.0.0`

---

## Out of scope (revisit after v1)

- Full IMS CSCF suite
- Built-in transcoding / conferencing
- Kamailio/OpenSIPS config converters as a **product promise**
- Proprietary softphone clients
- Replacing Traefik/APISIX as a general HTTP gateway

---

## How to influence the roadmap

1. Open a **Discussion** with the problem, not only a solution.
2. For resource schema or RFC changes, PR under `docs/design/`.
3. Interop captures (pcap) are gold — attach to Issues.
4. Map features to [gateway-patterns.md](docs/design/gateway-patterns.md) and/or an [RFC](docs/design/rfc/README.md).
5. Use [BACKLOG.md](docs/design/BACKLOG.md) IDs when promoting deferred work.

See [CONTRIBUTING.md](CONTRIBUTING.md).
