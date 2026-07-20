[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-P0%20done%20·%20P1%E2%80%93P4%20core-green.svg)](ROADMAP.md)
[![Go](https://img.shields.io/badge/go-1.23%2B-00ADD8.svg)](https://go.dev/)
[![SIP](https://img.shields.io/badge/SIP-RFC%203261-informational.svg)](https://datatracker.ietf.org/doc/html/rfc3261)

**面向云原生的 Go 语言 SIP 信令面（Signaling Plane）。**

sipplane 是开源的 **SIP 代理 / 注册服务器 / 边缘信令网关**，目标不是复刻 Kamailio / OpenSIPS 的本地 cfg，而是采用 **控制面 + 数据面** 架构：声明式策略、热更新、集群共享状态。

> **当前进度：** **P0（文档/RFC）已完成。** P1 可通话 MVP～P3 集群/发现核心已实现并通过测试；部分 P4 边缘能力（TLS、NAT/Path、HEP、Webhook、Helm、OTel 等）已提前落地。规格见 [docs/](docs/) 与 [ROADMAP.md](ROADMAP.md)。

English → [README.md](README.md)

---

## 快速开始

```bash
# 依赖：Go 1.23+（GOTOOLCHAIN=auto 可自动拉取工具链）
go test ./...
# 或完整自动化（含 Postgres/Redis/E2E）：
#   .\scripts\test.ps1 all          # Windows
#   ./scripts/test.sh all           # Linux/macOS
go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config
```

- 健康检查：`http://127.0.0.1:8080/readyz`
- 指标：`http://127.0.0.1:8080/metrics`
- 测试说明：[docs/testing.md](docs/testing.md)
- 控制面：[docs/control-plane.md](docs/control-plane.md)
- 策略：[docs/policies.md](docs/policies.md)
- 集群/发现：[docs/cluster.md](docs/cluster.md)
- 边缘能力（P4）：[docs/edge.md](docs/edge.md)
- 威胁模型：[docs/threat-model.md](docs/threat-model.md)
- 互通矩阵：[docs/interop/matrix.md](docs/interop/matrix.md)
- 生产参考部署：[docs/deploy-production.md](docs/deploy-production.md)
- 互通说明：[docs/interop/README.md](docs/interop/README.md)
- SIPp：[examples/sipp/README.md](examples/sipp/README.md)
- 更多示例：[examples/README.md](examples/README.md)

---

## 为什么做 sipplane？

| 传统 SIP 代理（Kamailio / OpenSIPS） | sipplane |
|--------------------------------------|----------|
| 本地 `.cfg` + 可选数据库 | **声明式资源** + Watch / 热更新 |
| 进程内 location / dialog | **外置集群状态**（如 Redis） |
| 以 reload 为中心的运维 | **带 revision 的控制面**（API / GitOps） |
| C 模块扩展 | **Go 嵌入 + gRPC / Wasm 插件** |
| 大规模生产验证充分 | 面向 Go / 云原生团队的现代体验 |

**我们是：** 高性能 SIP **信令**面（代理、注册、中继路由、边缘策略）。

**我们不是：** 媒体服务器、PBX 或完整 IMS 核心。媒体交给 RTPEngine、FreeSWITCH、LiveKit 等。

---

## 已实现能力（摘要）

| 版本 | 状态 | 重点 |
|------|------|------|
| **P0** | **已完成** | 愿景、架构、RFC 0001–0005、治理文档 |
| **v0.1** | **已完成** | Proxy + Registrar + Digest + LocationStore + Prometheus + SIPp/互通说明 |
| **v0.2a** | **已完成** | 控制面 REST（apply/dry-run/watch）+ Memory/Postgres + sipplanectl + DP Watcher |
| **v0.2b** | **已完成** | ACL / RateLimit（bootstrap `policies:`）+ 策略说明 |
| **v0.3** | **核心已完成** | Redis Location + loadBalance 算法 + OPTIONS + [集群文档](docs/cluster.md) |
| **v0.4+** | **部分完成** | TLS、NAT/Path/Outbound、HEP、Webhook、302、OTel、Helm — 见 [edge.md](docs/edge.md) |

关键默认 → [RFC 摘要](docs/design/rfc/README.zh-CN.md) · 延后/已提前项 → [BACKLOG](docs/design/BACKLOG.md)

---

## 二进制

| 命令 | 作用 |
|------|------|
| `cmd/sipplane` | 数据面 |
| `cmd/sipplane-control` | 控制面 API |
| `cmd/sipplanectl` | 配置 apply / dry-run 客户端 |

---

## 许可证

[Apache License 2.0](LICENSE)
