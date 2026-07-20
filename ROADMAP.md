# Roadmap

sipplane ships **design before code**. Dates are indicative; milestones gate on quality, not calendar pressure.

| Phase | Version | Theme | Status |
|-------|---------|-------|--------|
| P0 | — | Docs, governance, **critical RFCs** | **In progress** |
| P1 | v0.1.0 | Callable single-node MVP | Planned |
| P2a | v0.2.0 | Control plane hot reload (core) | Planned |
| P2b | v0.2.x | Policies + ctl polish | Planned |
| P3 | v0.3.0 | Cluster state + discovery | Planned |
| P4 | v0.4.x | Production edge hardening | Planned |
| GA | v1.0.0 | Stable API + interop matrix | Future |

**Critical defaults (accepted):** [docs/design/rfc/](docs/design/rfc/README.md)  
**Deferred features:** [docs/design/BACKLOG.md](docs/design/BACKLOG.md)

---

## P0 — Foundation (now)

- [x] Public vision (README)
- [x] Architecture draft
- [x] Resource model draft
- [x] Comparison / positioning
- [x] Gateway patterns draft (policy / observe / CP-DP / discovery)
- [x] **Critical RFCs 0001–0005** (affinity, revision, store, Record-Route, location cache)
- [x] Explicit backlog document
- [x] Apache-2.0 license
- [x] Contributing & security policy
- [ ] Community RFC: freeze remaining `v1alpha1` resource field names (non-conflicting with RFCs)
- [ ] CI skeleton (docs lint / markdown only until code exists)
- [ ] Logo / social preview (optional)

**Exit criteria:** Maintainers treat RFCs 0001–0005 as defaults; P1 scope agreed; “good first design issue” list open.

---

## P1 — v0.1.0 “Hello call”

**Goal:** `docker compose` + SIPp (or softphone) can REGISTER and complete a proxied call through sipplane.

### Include

- [ ] Go module + `cmd/sipplane` binary skeleton
- [ ] UDP (+ TCP) listen via sipgo
- [ ] Stateful proxy: INVITE, ACK, BYE, CANCEL, OPTIONS
- [ ] **`advertised_host` required** ([RFC 0004](docs/design/rfc/0004-record-route.md)); Via / Record-Route for **single-node / lab topology**
- [ ] Registrar with **`LocationStore` interface** + in-memory impl ([RFC 0005](docs/design/rfc/0005-location-cache.md))
- [ ] Digest auth (REGISTER; optional INVITE)
- [ ] Local YAML resources mapped 1:1 to [resource model](docs/design/resource-model.md)
- [ ] Prometheus metrics + `/healthz` `/readyz` (labels aligned with gateway-patterns draft)
- [ ] Structured access log (minimal fields: call_id, method, code, duration)
- [ ] `examples/sipp` and `examples/docker-compose`
- [ ] **SIPp matrix:** REGISTER, INVITE+ACK+BYE, **CANCEL/487 race**, Record-Route host assertion
- [ ] Interop notes: FreeSWITCH, Asterisk

### Explicitly defer

- TLS/WSS, Redis, multi-tenant enforcement, B2BUA, media, NAT/Path (see [BACKLOG](docs/design/BACKLOG.md) B3)

**Exit criteria:** Documented happy-path call; SIPp regression in CI including CANCEL; no known data-race in proxy path; `LocationStore` interface stable enough for P3 Redis.

---

## P2a — v0.2.0 Control plane (core)

**Goal:** Change a Route without restarting the data plane.

Keep this milestone **thin** — gateway UX without boiling the ocean.

- [ ] Management API (gRPC **or** REST — pick one primary) for Tenant / Endpoint / Trunk / Route
- [ ] **PostgreSQL** resource store ([RFC 0003](docs/design/rfc/0003-config-store.md))
- [ ] Watch / snapshot push with monotonic `revision` ([RFC 0002](docs/design/rfc/0002-config-revision.md))
- [ ] Atomic apply + rollback on validation failure
- [ ] **Validate / dry-run** endpoint before commit
- [ ] Minimal audit (who/when/revision)
- [ ] `sipplane-control` binary **or** dual-mode single binary
- [ ] Freeze structured access log field set (call_id, tenant, route, trunk, revision, code, duration)

**Exit criteria:** Two data-plane replicas converge on same revision within SLA; CP brief outage does not drop SIP (last-known-good); dry-run rejects invalid Route without bumping revision; stale sync flips `/readyz`.

---

## P2b — v0.2.x Policies & tooling

**Goal:** Ingress policies + operator ergonomics on top of a working control plane.

- [ ] Policy bindings: `acl` + `rate_limit` (ingress phase)
- [ ] Richer audit log
- [ ] `sipplanectl apply` (thin client)
- [ ] `number_transform` (optional; or stay on [BACKLOG B2](docs/design/BACKLOG.md))

**Exit criteria:** Documented policy examples; rate-limit metrics visible; apply from Git directory works.

---

## P3 — v0.3.0 Cluster + discovery

**Goal:** Kill one data-plane pod; registrations survive. Upstream discovery behaves like APISIX Upstream.

**Affinity default:** [RFC 0001](docs/design/rfc/0001-affinity.md) (Call-ID hash + shared location).

- [ ] Redis `LocationStore` + local cache ([RFC 0005](docs/design/rfc/0005-location-cache.md))
- [ ] Document LB / consistent-hash deployment example
- [ ] Node registration / health
- [ ] **DispatchGroup**: weighted / round-robin / **Call-ID consistent hash**
- [ ] **Active OPTIONS health checks** + passive outlier eject
- [ ] DNS SRV refresh for trunk destinations (optional behind flag)
- [ ] Basic multi-tenant key isolation in state store
- [ ] `circuit_breaker` policy on trunk selection
- [ ] Optional: full dialog store flag ([BACKLOG B6](docs/design/BACKLOG.md))

**Exit criteria:** HA demo in `examples/`; unhealthy trunk ejected without blackholing; fail-closed location errors; load test report published.

---

## P4 — v0.4.x Production edge

Prioritized backlog (order may change):

1. SIP TLS + WSS
2. NAT / Path / topology hiding ([BACKLOG B3](docs/design/BACKLOG.md) — promote earlier if needed)
3. RTPEngine control integration (external media)
4. HEP → Homer
5. Helm chart + example Kubernetes manifests
6. Kubernetes EndpointSlice discovery
7. Webhook / gRPC routing plugin (prefer before Wasm)
8. Wasm exploration
9. OpenTelemetry (control plane + sampled SIP)
10. Dashboard (optional; API-first remains)
11. Parallel fork ([BACKLOG B1](docs/design/BACKLOG.md)) when demanded

---

## v1.0.0 GA

- [ ] `sipplane.io/v1` API (no breaking changes without major version)
- [ ] Published interop matrix
- [ ] Threat model + security audit notes (**bring earlier if public SIP face**)
- [ ] Gateway-patterns checklist green
- [ ] RFCs 0001–0005 either upheld or explicitly superseded
- [ ] At least one production reference deployment

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
