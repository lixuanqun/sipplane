# sipplane 网关模式借鉴

> 状态：**草案**。英文主文档：[gateway-patterns.md](gateway-patterns.md)

sipplane 是 **SIP 信令面**，不是 HTTP API 网关。
我们主动学习成熟开源网关（APISIX、Kong、Traefik、Tyk、KrakenD、Easegress、Envoy/Istio、Caddy）的产品能力，并映射到 SIP 语义（事务、Dialog、REGISTER、Trunk）。

**原则：借运维模型与控制面体验；守 SIP 协议正确性。**

## 1. 借鉴总表

| 能力 | 参考网关 | sipplane 映射 |
|------|----------|---------------|
| 策略 / 插件链 | APISIX、Kong、Easegress、Envoy | 有序 SIP Filter 阶段 |
| 可观测性 | APISIX、Traefik、Envoy | 指标 + 结构化 access log + HEP |
| 控制 / 数据分离 | APISIX、Envoy xDS、Caddy Admin | Admin API + revision Watch |
| 自动服务发现 | Traefik、Istio、go-zero | Trunk / 后端发现 + 健康检查 |
| Upstream 健康与负载 | APISIX Upstream、Kong | `DispatchGroup` + OPTIONS / 熔断 |
| Consumer / 身份 | Kong、Tyk | `Endpoint` + `Trunk` 凭证 |
| 限流 / 配额 | APISIX、Tyk | 租户 / 中继 / IP / 终端维度 |
| 声明式 GitOps | KrakenD、Traefik CRD | `sipplanectl apply` |
| 热更新 | APISIX、Caddy | 原子快照替换，不停 SIP |

## 2. 策略引擎

### 2.1 Filter 链

```text
ingress → auth → routing → egress → async
```

| 阶段 | 作用 | 示例 |
|------|------|------|
| ingress | 尽早准入/整形 | IP ACL、方法 ACL、CPS、头字段规范化 |
| auth | 身份证明 | Digest、Trunk IP 信任、SIP TLS mTLS |
| routing | 决定下一跳 | Route 匹配、号码变换、LB、查注册 |
| egress | 出站准备 | Trunk 鉴权、Record-Route、拓扑隐藏 |
| async | 副作用 | CDR、Webhook、事件总线、HEP |

规则：策略是**资源**（或挂在 Route/Tenant/Trunk 上）；有 priority；失败模式明确（deny / continue / fallback）；热路径本地缓存；外部 webhook 必须硬超时 + 默认动作。

### 2.2 官方策略路线

`acl`、`rate_limit`、`digest_auth`、`metrics`、`access_log` → 早期；
`number_transform`、`circuit_breaker` → 中期；
`webhook_policy`、Wasm/gRPC 插件 → 后期。

### 2.3 不照搬 HTTP 网关的部分

- 不以 Path/Host 为唯一匹配模型
- 不能不处理 Dialog / 事务
- 不默认随意改 SDP（牵涉媒体面）

## 3. 可观测性

### 3.1 Prometheus（v0.1 起）

标签化：`method`、`tenant`、`route`、`trunk`、`code`；另有 `config_revision`、upstream 健康、策略延迟、限流拒绝计数。

### 3.2 结构化 access log

每事务：`call_id`、method、租户/路由/中继、`config_revision`、源/目的、响应码、耗时；可选 `policy_trace`。

### 3.3 Tracing / HEP / 探针

- OpenTelemetry：先控制面，SIP 采样后续
- HEP → Homer：v0.4（SIP 原生信令镜像）
- `/healthz` 常开；`/readyz` 在配置过期或关键上游全挂时失败

## 4. 控制面 / 数据面分离

| 关注点 | 控制面 | 数据面 |
|--------|--------|--------|
| 资源 CRUD | ✓ | ✗ |
| SIP I/O | ✗ | ✓ |
| 权威配置 | 存储 | 仅缓存快照 |
| Location / Dialog | ✗ | 状态面（Redis） |
| 控制面宕机 | — | **继续** last-known-good |

下发默认：**全量快照 + 单调 revision**；校验失败不 bump；dry-run/validate API；审计变更。

## 5. 自动服务发现

| 来源 | 阶段 |
|------|------|
| 静态 YAML / API | v0.1–v0.2 |
| `Trunk` / `DispatchGroup` | v0.2+ |
| DNS SRV | v0.3+ |
| K8s Endpoints | v0.4+ |

发现必须配合 **主动 OPTIONS 探活 + 被动熔断 + Call-ID 一致性哈希**；全不健康返回确定 503。

SIP 特有：REGISTER location 本身就是发现系统；多实例必须亲和或共享 dialog；UDP 不能只靠 L4 sticky。

## 6. 贡献者检查清单

1. 对应哪类网关能力？（策略 / 观测 / CP-DP / 发现）
2. 是否在 SIP 热路径？超时与失败模式？
3. 是资源、策略绑定，还是内建能力？
4. 用哪些指标/日志证明生效？
5. 与 HTTP 类比的 SIP 差异是什么？

## 7. 相关文档

[架构](../architecture.zh-CN.md) · [资源模型](resource-model.zh-CN.md) · [设计原则](principles.md) · [路线图](../../ROADMAP.md)
