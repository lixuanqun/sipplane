# Testing Guide

本仓库用 Go 标准测试 + 可选 Docker 依赖，方便本地与 CI 自动化。

## 快速开始

```bash
# 仅单元 / 进程内集成（不强制外部依赖）
go test ./... -count=1

# Windows
.\scripts\test.ps1 unit

# Linux / macOS / CI
./scripts/test.sh unit
```

## 完整自动化（含 Postgres + Redis）

```bash
# 启动依赖并跑全部 go test + 控制面 E2E
./scripts/test.ps1 all          # Windows
./scripts/test.sh all           # Unix

# 或分步
./scripts/test.ps1 integration
./scripts/test.ps1 e2e-control
```

`examples/docker-compose/docker-compose.test.yml` 会拉起：

| 服务 | 宿主机端口 |
|------|------------|
| Postgres 16 | `5433` |
| Redis 7 | `6380` |

环境变量（脚本会自动设置，也可覆盖）：

```text
SIPPLANE_DATABASE_URL=postgres://sipplane:sipplane@127.0.0.1:5433/sipplane?sslmode=disable
SIPPLANE_REDIS_ADDR=127.0.0.1:6380
```

若本机已有 Postgres/Redis，可：

```powershell
$env:SKIP_DOCKER="1"
$env:SIPPLANE_DATABASE_URL="postgres://..."
$env:SIPPLANE_REDIS_ADDR="127.0.0.1:6379"
.\scripts\test.ps1 integration
```

## 测试清单（按包）

| 包 | 文件 | 覆盖内容 |
|----|------|----------|
| `internal/auth` | `digest_test.go` | Digest challenge/verify |
| `internal/config` | `config_test.go` | advertised_host 校验 |
| `internal/resources` | `loader_test.go` | YAML 资源加载 |
| `internal/routing` | `engine_test.go` | Route 匹配 |
| `internal/location` | `memory_test.go`, `redis_test.go` | Location 内存 / Redis CRUD + TTL |
| `internal/controlplane/store` | `store_test.go`, `postgres_test.go` | Memory/Postgres apply/watch + 非法 apply + watch 超时 |
| `internal/controlplane/api` | `api_test.go` | REST apply/dry-run/watch |
| `internal/controlplane/watcher` | `watcher_test.go` | DP 配置拉取 |
| `internal/registrar` | `registrar_test.go` | Digest REGISTER + Outbound Path |
| `internal/proxy` | `proxy_test.go` | reject/lookup、CANCEL、**webhook 决策/超时回退** |
| `internal/accesslog` | `accesslog_test.go` | 结构化 access log 字段 |
| `internal/dataplane` | `dataplane_test.go`, `cancel_test.go`, `tls_test.go`, `options_tcp_test.go` | 通话流、CANCEL、TLS、**OPTIONS**、**TCP**、**metrics 断言** |
| `internal/nat` | `nat_test.go` | NAT Contact 改写 |
| `internal/outbound` | `outbound_test.go` | flow-token |
| `internal/policy` | `policy_test.go` | ACL / 限流 |
| `internal/discovery` | `dispatch_test.go`, `health_test.go` | DispatchGroup + **OPTIONS 探活/熔断** |
| `internal/hep` | `encode_test.go` | HEP3 编码 + **UDP Send** |
| `internal/webhook` | `webhook_test.go` | webhook 超时回退 |
| `internal/transform` | `transform_test.go` | 号变换 |
| `internal/redirect` | `redirect_test.go` | 302 策略 |
| `internal/otelx` | `otelx_test.go` | OTel disabled + attrs |

> 测试均放在对应包旁的 `*_test.go`（无独立 `test/` 目录）。自动化脚本见 `scripts/`。

## SIPp / interop

- SIPp scenarios: [examples/sipp/README.md](../examples/sipp/README.md)
- **Automated SIPp smoke** (OPTIONS + Digest REGISTER):

```bash
make test-sipp
# or: bash scripts/sipp-smoke.sh / powershell scripts/sipp-smoke.ps1
```

Skips locally if `sipp` is missing; CI installs `sip-tester` and runs the job.

- **Compose smoke** (validate + Postgres/Redis probe):

```bash
make test-compose
```

- FreeSWITCH / Asterisk notes: [docs/interop/README.md](interop/README.md)
- Control plane: [control-plane.md](control-plane.md)
- Policies: [policies.md](policies.md)
- Cluster / discovery: [cluster.md](cluster.md)
- Edge (P4): [edge.md](edge.md)
- Threat model: [threat-model.md](threat-model.md)
- Interop matrix: [interop/matrix.md](interop/matrix.md)

## Makefile

```bash
make test              # unit
make test-integration  # 尽量拉起 docker 依赖后测试
make test-e2e          # 控制面 e2e（调用 scripts）
make test-sipp         # SIPp OPTIONS + REGISTER
make test-compose      # docker-compose.test 冒烟
make deps-up / deps-down
make build
```

## CI

见 `.github/workflows/test.yml`：

| Job | 内容 |
|-----|------|
| `unit` | `go test ./...` |
| `integration` | Postgres/Redis + `go test` + control e2e |
| `sipp` | 安装 SIPp + OPTIONS/REGISTER smoke |
| `compose` | `docker-compose.test.yml` up + probe |