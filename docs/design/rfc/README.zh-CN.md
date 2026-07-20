# 关键力 RFC 摘要（已采纳默认）

英文全文见同目录 `0001`–`0005`。在写代码前按下列默认实现，变更需 Discussion + 更新 RFC。

| RFC | 主题 | 默认决策 |
|-----|------|----------|
| [0001](0001-affinity.md) | 呼叫亲和 | Call-ID 一致性哈希 + 共享 Location；全量 Dialog 进 Redis 为可选项 |
| [0002](0002-config-revision.md) | 配置下发 | 全量快照 + 单调 revision；超时未同步 → `/readyz` 失败；CP 宕机用 last-known-good |
| [0003](0003-config-store.md) | 配置存储 | **PostgreSQL** 存资源与审计；Watch 用 NOTIFY/轮询；etcd 可选非必须 |
| [0004](0004-record-route.md) | Record-Route | 只宣告 `advertised_host`（VIP/DNS），禁止仅用 Pod IP；缺省配置则拒绝启动 |
| [0005](0005-location-cache.md) | Location 读路径 | P1 抽象 `LocationStore`；P3 Redis + 本地短缓存；查找失败 **fail-closed**（503/480） |

索引：[README.md](README.md)
