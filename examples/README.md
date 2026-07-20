# sipplane examples

## Lab config

| File | Purpose |
|------|---------|
| `config/bootstrap.yaml` | Data-plane listen / `advertised_host` / HTTP |
| `config/lab.yaml` | Tenant / Endpoint / Trunk / Route resources |

## Run data plane (P1 happy path)

```bash
go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config
```

- Health: http://127.0.0.1:8080/readyz
- Metrics: http://127.0.0.1:8080/metrics

Then either:

- Softphone REGISTER as `alice` / `alice-secret` (realm `sipplane`), or
- SIPp: [sipp/README.md](sipp/README.md), or
- Go e2e: `go test ./internal/dataplane -run TestHealthAndRegisterInviteFlow -v`

## Control plane + hot reload

See [docs/control-plane.md](../docs/control-plane.md).

```bash
# terminal 1 — control plane (Memory store; use -database-url for Postgres)
go run ./cmd/sipplane-control -listen 127.0.0.1:8090 -seed examples/config

# terminal 2 — data plane watching CP
# set control_url: http://127.0.0.1:8090 in bootstrap or:
#   set SIPPLANE_CONTROL_URL=http://127.0.0.1:8090
go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config

# apply change
go run ./cmd/sipplanectl --server http://127.0.0.1:8090 apply examples/config/lab.yaml
go run ./cmd/sipplanectl --server http://127.0.0.1:8090 revision
```

## Ingress policies (ACL / rate limit)

See [docs/policies.md](../docs/policies.md). Uncomment `policies:` in `config/bootstrap.yaml`.

## Cluster / load balance (P3)

- Guide: [docs/cluster.md](../docs/cluster.md)
- Lab resources: [config/lab-lb.yaml](config/lab-lb.yaml) (`loadBalance` + `consistent_hash` + OPTIONS ping)
- Front LB sketches: [deploy/](deploy/)

```bash
# Redis location
# set redis_addr / SIPPLANE_REDIS_ADDR then:
go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config
```

## Edge features (P4)

- Guide: [docs/edge.md](../docs/edge.md)
- Bootstrap sketch: [config/bootstrap-edge.yaml](config/bootstrap-edge.yaml)
- Helm: [deploy/helm/sipplane/README.md](../deploy/helm/sipplane/README.md)

## Docker Compose

```bash
cd examples/docker-compose
docker compose up --build
```

Test dependencies only (Postgres `:5433`, Redis `:6380`):

```bash
docker compose -f docker-compose.test.yml up -d --wait
```

See [docs/testing.md](../docs/testing.md).

## SIPp

[sipp/](sipp/) — OPTIONS, Digest REGISTER; INVITE/CANCEL via Go tests.

## Interop

[docs/interop/README.md](../docs/interop/README.md) — FreeSWITCH / Asterisk / softphone notes.
