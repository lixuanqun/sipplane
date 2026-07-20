# RFC index — critical defaults

> Status: **Accepted defaults for implementation** (may revise via Discussion before v1).
>
> These RFCs close open questions from the architecture review so P1+ code does not guess.

| RFC | Title | Default decision |
|-----|-------|------------------|
| [0001](0001-affinity.md) | Dialog / call affinity | Call-ID consistent hash + shared location; full dialog store optional |
| [0002](0002-config-revision.md) | Config distribution | Full snapshot + monotonic revision; stale → not ready |
| [0003](0003-config-store.md) | Config persistence | PostgreSQL for resources; channel for Watch notify |
| [0004](0004-record-route.md) | Record-Route / Contact | Advertise VIP / advertised host, never Pod-only IP |
| [0005](0005-location-cache.md) | Location read path | Local TTL cache over Redis; fail-closed on lookup timeout |

中文摘要：[README.zh-CN.md](README.zh-CN.md)

## Process

1. These are **recommended defaults**, not forever law.
2. Breaking a default before `v1` requires a short Discussion + PR updating the RFC.
3. Implementation MUST document deviations in release notes.
