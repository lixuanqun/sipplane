# Roadmap

sipplane ships **design before code**. Dates are indicative; milestones gate on quality, not calendar pressure.

| Phase | Version | Theme | Status |
|-------|---------|-------|--------|
| P0 | — | Docs, governance, GitHub presence | **In progress** |
| P1 | v0.1.0 | Callable single-node MVP | Planned |
| P2 | v0.2.0 | Control plane + hot reload | Planned |
| P3 | v0.3.0 | Cluster state + HA | Planned |
| P4 | v0.4.x | Production edge hardening | Planned |
| GA | v1.0.0 | Stable API + interop matrix | Future |

---

## P0 — Foundation (now)

- [x] Public vision (README)
- [x] Architecture draft
- [x] Resource model draft
- [x] Comparison / positioning
- [x] Apache-2.0 license
- [x] Contributing & security policy
- [ ] Community RFC: freeze v1alpha1 resource field names
- [ ] CI skeleton (docs lint / markdown only until code exists)
- [ ] Logo / social preview (optional)

**Exit criteria:** Maintainers agree P1 scope; open “good first design issue” list.

---

## P1 — v0.1.0 “Hello call”

**Goal:** `docker compose` + SIPp (or softphone) can REGISTER and complete a proxied call through sipplane.

### Include

- [ ] Go module + `cmd/sipplane` binary skeleton
- [ ] UDP (+ TCP) listen via sipgo
- [ ] Stateful proxy: INVITE, ACK, BYE, CANCEL, OPTIONS
- [ ] Via / Record-Route handling for basic topologies
- [ ] Registrar with in-memory location store
- [ ] Digest auth (REGISTER; optional INVITE)
- [ ] Local YAML resources mapped to [resource model](docs/design/resource-model.md)
- [ ] Prometheus metrics + `/healthz` `/readyz`
- [ ] `examples/sipp` and `examples/docker-compose`
- [ ] Interop notes: FreeSWITCH, Asterisk

### Explicitly defer

- TLS/WSS, Redis, multi-tenant enforcement, B2BUA, media

**Exit criteria:** Documented happy-path call; SIPp regression in CI; no known data-race in proxy path.

---

## P2 — v0.2.0 Control plane

**Goal:** Change a Route without restarting the data plane.

- [ ] Management API (gRPC and/or REST) for Tenant / Endpoint / Trunk / Route
- [ ] Durable config store (PostgreSQL **or** etcd — decision in RFC)
- [ ] Watch / snapshot push with `revision`
- [ ] Atomic apply + rollback on validation failure
- [ ] Audit log of config changes
- [ ] `sipplane-control` binary (or dual-mode single binary)

**Exit criteria:** Two data-plane replicas receive the same revision within SLA; chaos test: control plane brief outage does not drop SIP.

---

## P3 — v0.3.0 Cluster

**Goal:** Kill one data-plane pod; registrations and in-flight signaling remain correct.

- [ ] Redis location backend (TTL-aligned with Expires)
- [ ] Affinity strategy documented + implemented (hash and/or shared dialog)
- [ ] Node registration / health
- [ ] DispatchGroup + OPTIONS probing
- [ ] Basic multi-tenant key isolation in state store

**Exit criteria:** HA demo script in `examples/`; load test report published.

---

## P4 — v0.4.x Production edge

Prioritized backlog (order may change):

1. SIP TLS + WSS
2. NAT / Path / topology hiding helpers
3. RTPEngine control integration (external media)
4. HEP → Homer
5. Helm chart + example Kubernetes manifests
6. Webhook / gRPC routing plugin
7. RateLimit / ACL resources
8. OpenTelemetry

---

## v1.0.0 GA

- [ ] `sipplane.io/v1` API (no breaking changes without major version)
- [ ] Published interop matrix (vendors / softswitches / WebRTC gateways)
- [ ] Threat model + security audit notes
- [ ] At least one production reference deployment (public or anonymized)

---

## Out of scope (revisit after v1)

- Full IMS CSCF suite
- Built-in transcoding / conferencing
- Kamailio/OpenSIPS config converters as a product promise
- Proprietary softphone clients

---

## How to influence the roadmap

1. Open a **Discussion** with the problem, not only a solution.
2. For resource schema changes, propose a short RFC in `docs/design/`.
3. Interop captures (pcap) are gold — attach to Issues.

See [CONTRIBUTING.md](CONTRIBUTING.md).
