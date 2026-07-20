# sipplane

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-design%20phase-orange.svg)](ROADMAP.md)
[![Go](https://img.shields.io/badge/go-1.22%2B-00ADD8.svg)](https://go.dev/)
[![SIP](https://img.shields.io/badge/SIP-RFC%203261-informational.svg)](https://datatracker.ietf.org/doc/html/rfc3261)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**Cloud-native SIP signaling plane for Go.**

sipplane is an open-source **SIP proxy / registrar / edge signaling gateway** designed for Kubernetes-era telephony — not a rewrite of Kamailio or OpenSIPS config scripts, but a **control-plane + data-plane** architecture with hot-reloadable policies and cluster-shared state.

> **Current phase: architecture & planning.** Implementation has not started yet. Specs are open for review — see [docs/](docs/) and [ROADMAP.md](ROADMAP.md). Contributions to design discussions are welcome.

中文说明 → [README.zh-CN.md](README.zh-CN.md)

---

## Why sipplane?

| Traditional SIP proxies (Kamailio / OpenSIPS) | sipplane |
|-----------------------------------------------|----------|
| Local `.cfg` scripts + optional DB | **Declarative resources** + Watch / hot update |
| In-process location / dialog memory | **Externalized cluster state** (e.g. Redis) |
| Reload-centric operations | **Revisioned control plane** (API / GitOps-ready) |
| C modules for extension | **Go embed + gRPC / Wasm plugins** |
| Battle-tested at scale | Modern DX for Go / cloud teams |

**What we are:** a high-performance SIP **signaling** plane (proxy, registrar, trunk routing, edge policies).

**What we are not:** a media server, PBX, or full IMS core. Media stays with RTPEngine, FreeSWITCH, LiveKit, or similar.

---

## Positioning

```text
┌─────────────────────────────────────────────────────────────┐
│                     Control Plane                            │
│   Trunks · Routes · Endpoints · Tenants · ACL · Revisions   │
│              REST / gRPC  ·  Watch  ·  Audit                 │
└────────────────────────────┬────────────────────────────────┘
                             │ push / subscribe
┌────────────────────────────▼────────────────────────────────┐
│                      Data Plane                              │
│         Stateful Proxy · Registrar · Auth · LB              │
│              (built on sipgo — no stack rewrite)             │
└──────────────┬───────────────────────────┬──────────────────┘
               │                           │
        ┌──────▼──────┐             ┌──────▼──────┐
        │ Shared State │             │   Events    │
        │ Redis / etcd │             │ NATS/Kafka  │
        └─────────────┘             └─────────────┘
```

**Stack relationship**

| Layer | Project | Role |
|-------|---------|------|
| SIP stack | [sipgo](https://github.com/emiago/sipgo) | Parse, transport, transactions |
| Dialog / media helpers (optional) | [diago](https://github.com/emiago/diago) | When B2BUA / local RTP is needed later |
| **This project** | **sipplane** | Platform: routing, registrar, control plane, cluster |

---

## Planned capabilities

### v0.1 — Callable MVP
- UDP/TCP SIP listener
- Stateful proxy (INVITE / ACK / BYE / CANCEL)
- Registrar (in-memory location, pluggable store interface)
- Digest authentication
- Static declarative routes (YAML/JSON)
- Metrics (Prometheus) + health endpoints
- SIPp examples + FreeSWITCH / Asterisk interop notes

### v0.2 — Control plane (split)
- **P2a:** Management API + PostgreSQL + Watch/revision + dry-run
- **P2b:** ACL / rate limit + `sipplanectl`

### v0.3 — Cluster + discovery
- Redis location + local cache (fail-closed)
- Call-ID consistent hash affinity ([RFC 0001](docs/design/rfc/0001-affinity.md))
- DispatchGroup: health checks, outlier eject

### v0.4+ — Production edge
- TLS / WSS, NAT / Path, RTPEngine, HEP, Helm, K8s discovery, plugins

**Critical defaults** → [docs/design/rfc/](docs/design/rfc/README.md) · **Deferred** → [BACKLOG](docs/design/BACKLOG.md)

**Gateway-grade patterns** → [docs/design/gateway-patterns.md](docs/design/gateway-patterns.md)

Full detail: **[ROADMAP.md](ROADMAP.md)** · **[docs/architecture.md](docs/architecture.md)** · **[docs/design/resource-model.md](docs/design/resource-model.md)**

---

## Status

| Area | Status |
|------|--------|
| Vision & architecture | Draft — open for review |
| Resource / API model | Draft |
| Implementation (`cmd/`, `pkg/`) | **Not started** |
| Releases | None yet |

We publish design first on purpose: better APIs, clearer contribution surface, and a narrative developers can trust before code lands.

---

## Documentation

| Doc | Description |
|-----|-------------|
| [Architecture](docs/architecture.md) | Control / data / state planes |
| [Gateway patterns](docs/design/gateway-patterns.md) | Learn from APISIX / Traefik / Tyk / Easegress … |
| [Critical RFCs](docs/design/rfc/README.md) | Affinity, revision, store, Record-Route, location |
| [Resource model](docs/design/resource-model.md) | Trunk, Route, Endpoint, … |
| [Backlog](docs/design/BACKLOG.md) | Deferred features (fork, NAT, …) |
| [Comparison](docs/comparison.md) | vs Kamailio, OpenSIPS, sipgo, LiveKit SIP |
| [Roadmap](ROADMAP.md) | Phased milestones (P2a/P2b split) |
| [Contributing](CONTRIBUTING.md) | How to help (design PRs welcome now) |
| [Security](SECURITY.md) | Vulnerability reporting |

---

## Who is this for?

- Teams building **CPaaS / UCaaS / SIP trunking** who want Go + Kubernetes-native ops
- Engineers tired of maintaining fragile multi-node `.cfg` drift
- Contributors who know SIP **or** distributed systems — both are valuable here
- Projects that need a **signaling edge** in front of FreeSWITCH, Asterisk, LiveKit, or custom media

---

## License

[Apache License 2.0](LICENSE)

---

## Community

- **Issues** — design questions, RFCs, interop reports
- **Discussions** — architecture debates (preferred before large PRs)
- **PRs** — docs and design first; code once P1 milestones open

Star the repo if cloud-native SIP in Go matters to you — it helps others find the project during the design phase.
