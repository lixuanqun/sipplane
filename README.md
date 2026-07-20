# sipplane

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-P0%20done%20·%20P1%E2%80%93P4%20core-green.svg)](ROADMAP.md)
[![Go](https://img.shields.io/badge/go-1.23%2B-00ADD8.svg)](https://go.dev/)
[![SIP](https://img.shields.io/badge/SIP-RFC%203261-informational.svg)](https://datatracker.ietf.org/doc/html/rfc3261)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

**Cloud-native SIP signaling plane for Go.**

sipplane is an open-source **SIP proxy / registrar / edge signaling gateway** designed for Kubernetes-era telephony — not a rewrite of Kamailio or OpenSIPS config scripts, but a **control-plane + data-plane** architecture with hot-reloadable policies and cluster-shared state.

> **Current status:** **P0 (docs/RFCs) is done.** P1 callable MVP through P3 cluster/discovery core are implemented and tested; selected P4 edge features (TLS, NAT/Path, HEP, webhook, Helm, OTel, …) landed ahead of schedule. See [docs/](docs/) and [ROADMAP.md](ROADMAP.md).

中文说明 → [README.zh-CN.md](README.zh-CN.md)

---

## Quick start

```bash
# Requires Go 1.23+ (GOTOOLCHAIN=auto pulls toolchain if needed)
go test ./...
# Full automation (Postgres/Redis/E2E):
#   ./scripts/test.sh all          # Linux/macOS
#   .\scripts\test.ps1 all         # Windows

go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config
```

- Health: `http://127.0.0.1:8080/readyz`
- Metrics: `http://127.0.0.1:8080/metrics`
- Testing guide: [docs/testing.md](docs/testing.md)
- Examples: [examples/README.md](examples/README.md)

### Binaries

| Command | Role |
|---------|------|
| `cmd/sipplane` | Data plane |
| `cmd/sipplane-control` | Control-plane API |
| `cmd/sipplanectl` | apply / dry-run client |

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
        │ Redis        │             │ NATS/Kafka  │
        └─────────────┘             └─────────────┘
```

**Stack relationship**

| Layer | Project | Role |
|-------|---------|------|
| SIP stack | [sipgo](https://github.com/emiago/sipgo) | Parse, transport, transactions |
| Dialog / media helpers (optional) | [diago](https://github.com/emiago/diago) | When B2BUA / local RTP is needed later |
| **This project** | **sipplane** | Platform: routing, registrar, control plane, cluster |

---

## Capabilities (by phase)

| Phase | Status | Highlights |
|-------|--------|------------|
| **P0** | **Done** | Vision, architecture, RFCs 0001–0005, governance |
| **v0.1 (P1)** | **Done** | Proxy + Registrar + Digest + LocationStore + Prometheus + SIPp/interop notes |
| **v0.2a (P2a)** | **Done** | Control REST (apply/dry-run/watch) + Memory/Postgres + sipplanectl + DP Watcher |
| **v0.2b (P2b)** | **Done** | ACL / RateLimit via bootstrap `policies:` + cookbook |
| **v0.3 (P3)** | **Done (core)** | Redis Location + loadBalance algorithms + OPTIONS + [cluster docs](docs/cluster.md) |
| **v0.4+ (P4)** | **Partial** | TLS, NAT/Path/Outbound, HEP, Webhook, 302, OTel, Helm — see [edge.md](docs/edge.md) |

**Critical defaults** → [docs/design/rfc/](docs/design/rfc/README.md) · **Deferred** → [BACKLOG](docs/design/BACKLOG.md)

**Gateway-grade patterns** → [docs/design/gateway-patterns.md](docs/design/gateway-patterns.md)

Full detail: **[ROADMAP.md](ROADMAP.md)** · **[docs/architecture.md](docs/architecture.md)** · **[docs/design/resource-model.md](docs/design/resource-model.md)**

---

## Status

| Area | Status |
|------|--------|
| Vision & architecture | **Accepted** (P0 done) |
| Critical RFCs 0001–0005 | **Accepted** — implemented defaults |
| Resource / API model | `v1alpha1` (field freeze still open) |
| Implementation (`cmd/`, `internal/`) | **Active** — P1–P3 core + selected P4 |
| Automated tests | `go test ./...` + [docs/testing.md](docs/testing.md) |
| Releases | pre-1.0 (no GA tag yet) |

---

## Documentation

| Doc | Description |
|-----|-------------|
| [Architecture](docs/architecture.md) | Control / data / state planes |
| [Testing](docs/testing.md) | Automated tests, scripts, Docker deps |
| [Control plane](docs/control-plane.md) | REST API, sipplanectl, Watch / dry-run |
| [Policies](docs/policies.md) | ACL / rate-limit cookbook |
| [Cluster / discovery](docs/cluster.md) | Redis location, Call-ID affinity, loadBalance |
| [Edge (P4)](docs/edge.md) | TLS, NAT/Path, HEP, Webhook, redirect, OTel, Helm |
| [Threat model](docs/threat-model.md) | Pre-1.0 security draft |
| [Interop matrix](docs/interop/matrix.md) | Pass/Fail tracker toward GA |
| [Production deploy](docs/deploy-production.md) | Helm reference topology + NetworkPolicy |
| [Interop](docs/interop/README.md) | FreeSWITCH / Asterisk / softphone notes |
| [SIPp examples](examples/sipp/README.md) | OPTIONS + Digest REGISTER scenarios |
| [Gateway patterns](docs/design/gateway-patterns.md) | Learn from APISIX / Traefik / Tyk / Easegress … |
| [Gateway checklist](docs/design/gateway-checklist.md) | Pre-GA pattern tracker |
| [Critical RFCs](docs/design/rfc/README.md) | Affinity, revision, store, Record-Route, location |
| [Resource model](docs/design/resource-model.md) | Trunk, Route, Endpoint, … |
| [Backlog](docs/design/BACKLOG.md) | Deferred / promoted features |
| [Comparison](docs/comparison.md) | vs Kamailio, OpenSIPS, sipgo, LiveKit SIP |
| [Roadmap](ROADMAP.md) | Phased milestones |
| [Contributing](CONTRIBUTING.md) | How to help |
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

- **Issues** — bugs, design questions, RFCs, interop reports ([good first issues](https://github.com/lixuanqun/sipplane/labels/good%20first%20issue) welcome)
- **Discussions** — architecture debates (preferred before large design PRs)
- **PRs** — docs **and** code; follow [CONTRIBUTING.md](CONTRIBUTING.md) and accepted RFCs

Star the repo if cloud-native SIP in Go matters to you.
