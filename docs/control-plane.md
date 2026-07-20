# Control plane (P2a)

REST API served by `cmd/sipplane-control`. Client: `cmd/sipplanectl`.

## Quick start

```bash
# Memory store + seed lab YAML
go run ./cmd/sipplane-control -listen 127.0.0.1:8090 -seed examples/config

# With Bearer auth (recommended outside lab)
# export SIPPLANE_CONTROL_TOKEN=dev-secret
# go run ./cmd/sipplane-control -listen 127.0.0.1:8090 -seed examples/config -auth-token "$SIPPLANE_CONTROL_TOKEN"

# Postgres (optional)
# go run ./cmd/sipplane-control -listen 127.0.0.1:8090 \
#   -database-url 'postgres://sipplane:sipplane@127.0.0.1:5433/sipplane?sslmode=disable' \
#   -seed examples/config
```

Data plane watching the control plane:

```bash
# bootstrap: control_url: http://127.0.0.1:8090
# or: set SIPPLANE_CONTROL_URL=http://127.0.0.1:8090
# if CP uses a token: SIPPLANE_CONTROL_TOKEN=dev-secret (or control_token in bootstrap)
go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config
```

Apply / dry-run:

```bash
go run ./cmd/sipplanectl --server http://127.0.0.1:8090 dry-run examples/config/lab.yaml
go run ./cmd/sipplanectl --server http://127.0.0.1:8090 apply examples/config/lab.yaml
go run ./cmd/sipplanectl --server http://127.0.0.1:8090 revision
go run ./cmd/sipplanectl --server http://127.0.0.1:8090 snapshot

# With auth:
# go run ./cmd/sipplanectl --server http://127.0.0.1:8090 --token "$SIPPLANE_CONTROL_TOKEN" apply examples/config/lab.yaml
```

Automated e2e: `.\scripts\test.ps1 e2e-control` / `./scripts/test.sh e2e-control`.

## Authentication

| Setting | Behavior |
|---------|----------|
| `-auth-token` / `SIPPLANE_CONTROL_TOKEN` empty | Open API (lab only) |
| Non-empty token | All `/v1/*` require `Authorization: Bearer <token>` |
| `GET /healthz` | Always anonymous (probes) |

This is a **single shared token** (not RBAC / multi-identity). Treat it like a deploy secret; keep CP off the public Internet. mTLS / roles are future work ([threat-model](threat-model.md) §7).

Clients:

| Client | How to pass token |
|--------|-------------------|
| `sipplanectl` | `--token` or `SIPPLANE_CONTROL_TOKEN` |
| Data-plane watcher | `control_token` / `SIPPLANE_CONTROL_TOKEN` |

## HTTP API

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/healthz` | Liveness (no auth) |
| `GET` | `/v1/revision` | `{"revision":N}` |
| `GET` | `/v1/snapshot` | Full resource snapshot JSON |
| `GET` | `/v1/watch?since=N&timeout=30s` | Long-poll until revision > N (or timeout) |
| `POST` | `/v1/dry-run` | Validate YAML body; **does not** bump revision |
| `POST` | `/v1/apply` | Validate + commit; bumps revision; writes audit |
| `GET` | `/v1/audit?limit=50` | Recent apply audit entries |

- Body for apply/dry-run: `Content-Type: application/yaml` (multi-doc YAML OK).
- Optional header: `X-Actor: sipplanectl` (audit who).
- When auth enabled: `Authorization: Bearer <token>`.

## Behavior (RFC 0002 / 0003)

| Rule | Behavior |
|------|----------|
| Dry-run | Validation only; revision unchanged |
| Apply validation failure | HTTP 400; revision unchanged |
| Watch | Memory: channel notify; Postgres: LISTEN/NOTIFY + poll |
| DP stale | If no successful sync within `config_stale_after` (default 60s), `/readyz` → 503 |
| CP outage | Data plane keeps **last-known-good** snapshot |

## Stores

| Flag | Store |
|------|-------|
| (default) | In-memory |
| `-database-url` / `SIPPLANE_DATABASE_URL` | PostgreSQL |

## Related

- Policies (ACL / rate limit): [policies.md](policies.md)
- Threat model: [threat-model.md](threat-model.md)
- Testing: [testing.md](testing.md)
- RFCs: [0002](design/rfc/0002-config-revision.md), [0003](design/rfc/0003-config-store.md)
