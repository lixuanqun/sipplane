# sipplane

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-设计阶段-orange.svg)](ROADMAP.md)
[![Go](https://img.shields.io/badge/go-1.22%2B-00ADD8.svg)](https://go.dev/)
[![SIP](https://img.shields.io/badge/SIP-RFC%203261-informational.svg)](https://datatracker.ietf.org/doc/html/rfc3261)

**面向云原生的 Go 语言 SIP 信令面（Signaling Plane）。**

sipplane 是开源的 **SIP 代理 / 注册服务器 / 边缘信令网关**，目标不是复刻 Kamailio / OpenSIPS 的本地 cfg，而是采用 **控制面 + 数据面** 架构：声明式策略、热更新、集群共享状态。

> **当前阶段：架构与功能规划。** 业务代码尚未开始。规格开放评审 —— 见 [docs/](docs/) 与 [ROADMAP.md](ROADMAP.md)。欢迎参与设计讨论。

English → [README.md](README.md)

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

## 定位

```text
┌─────────────────────────────────────────────────────────────┐
│                         控制面                                │
│   Trunk · Route · Endpoint · Tenant · ACL · Revision        │
│              REST / gRPC  ·  Watch  ·  审计                   │
└────────────────────────────┬────────────────────────────────┘
                             │ 推送 / 订阅
┌────────────────────────────▼────────────────────────────────┐
│                         数据面                                │
│         Stateful Proxy · Registrar · Auth · LB              │
│              （基于 sipgo，不重写协议栈）                        │
└──────────────┬───────────────────────────┬──────────────────┘
               │                           │
        ┌──────▼──────┐             ┌──────▼──────┐
        │   共享状态    │             │    事件      │
        │ Redis / etcd │             │ NATS/Kafka  │
        └─────────────┘             └─────────────┘
```

| 层级 | 项目 | 职责 |
|------|------|------|
| SIP 栈 | [sipgo](https://github.com/emiago/sipgo) | 解析、传输、事务 |
| Dialog / 媒体（可选） | [diago](https://github.com/emiago/diago) | 后续 B2BUA / 本地 RTP |
| **本项目** | **sipplane** | 平台：路由、注册、控制面、集群 |

---

## 规划能力（摘要）

| 版本 | 重点 |
|------|------|
| **v0.1** | 可联调 Proxy + Registrar + LocationStore 接口 + SIPp（含 CANCEL） |
| **v0.2a** | PostgreSQL 控制面 + Watch/revision + dry-run |
| **v0.2b** | ACL/限流 + sipplanectl |
| **v0.3** | Redis Location + Call-ID 亲和 + DispatchGroup 探活 |
| **v0.4+** | TLS/WSS、NAT、RTPEngine、HEP、Helm、插件 |

关键默认 → [RFC 摘要](docs/design/rfc/README.zh-CN.md) · 延后项 → [BACKLOG](docs/design/BACKLOG.md)

详情：[ROADMAP.md](ROADMAP.md) · [架构](docs/architecture.zh-CN.md) · [资源模型](docs/design/resource-model.zh-CN.md)

---

## 文档

| 文档 | 说明 |
|------|------|
| [架构设计](docs/architecture.zh-CN.md) | 控制面 / 数据面 / 状态面 |
| [网关模式借鉴](docs/design/gateway-patterns.zh-CN.md) | 学 APISIX / Traefik / Tyk / Easegress … |
| [关键 RFC](docs/design/rfc/README.zh-CN.md) | 亲和、revision、存储、RR、Location |
| [资源模型](docs/design/resource-model.zh-CN.md) | Trunk、Route、Endpoint 等 |
| [Backlog](docs/design/BACKLOG.md) | 延后功能（fork、NAT 等） |
| [对比说明](docs/comparison.zh-CN.md) | 与 Kamailio、OpenSIPS、sipgo 等 |
| [路线图](ROADMAP.md) | 分阶段里程碑（P2a/P2b） |
| [贡献指南](CONTRIBUTING.md) | 当前欢迎设计类贡献 |

---

## 许可证

[Apache License 2.0](LICENSE)

欢迎 Star，帮助更多开发者在设计阶段发现本项目。
