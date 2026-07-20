# 资源模型

> 状态：**草案**。英文主文档：[resource-model.md](resource-model.md)

sipplane 的配置是一组**带版本的资源**，不是单体脚本。每次生效的配置集对应数据面可见的单调 `revision`。

## 通用元数据

```yaml
apiVersion: sipplane.io/v1alpha1
kind: Route
metadata:
  name: pstn-egress
  tenant: acme
spec: {}
```

## 资源一览

| Kind | 作用 |
|------|------|
| `Tenant` | 租户隔离、配额 |
| `Endpoint` | 可注册 / 可主叫的 UA 或 PBX |
| `Trunk` | 对接运营商 / SBC / 对端平台 |
| `Route` | 匹配 → 动作（代理、负载均衡、查注册、拒绝） |
| `DispatchGroup` | 后端组 + 健康检查（v0.3+） |
| `ACL` / `RateLimit` | 安全与限流 |

凭证一律 `passwordSecretRef`，禁止把密码写进 Git。

## 生命周期

`API 写入 → 校验 → 存储 → revision++ → Watch 通知 → 数据面原子换快照`

v0.1 可用本地 YAML（字段与本模型一致），v0.2 起以控制面 API 为准。

## 刻意不做的

- 媒体 / 编解码配置（媒体面适配器）
- 完整拨号计划语言（复杂逻辑走 webhook / 插件）
- Kamailio 脚本函数兼容层
