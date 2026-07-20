# RFC 0002 — Config revision & distribution

- **Status:** Accepted default
- **Target:** P2a
- **Related:** architecture §4; gateway-patterns §4.2

## Problem

Data planes must apply the same policy without restart, survive control-plane outages, and avoid split-brain revisions.

## Decision

| Rule | Default |
|------|---------|
| Unit of distribution | **Full configuration snapshot** (all tenants visible to that DP, or global) |
| Version | Monotonic uint64 **`revision`** per control plane |
| Transport | Watch / long-poll / gRPC stream — implementation choice; semantics identical |
| Apply | **Atomic swap** of in-memory snapshot; readers see old or new, never mix |
| Validate | Reject bad snapshot **before** bumping revision; keep previous |
| CP outage | DP continues on **last-known-good** |
| Stale SLA | If no successful sync for **`config_stale_after`** (default **60s**), set **`/readyz` = false** (leave existing calls; stop taking new load via LB) |
| Incremental xDS | **Deferred** after v0.3; not required for correctness |

```text
apply request → validate → commit revision=N → notify DPs
DP: fetch snapshot N → swap → gauge config_revision=N
```

## Single-node (P1)

- Load YAML → treat as revision `1` (or file mtime hash).
- No Watch required; SIGHUP or file watch optional.

## Metrics

- `sipplane_config_revision` (gauge)
- `sipplane_config_apply_total{result="ok|error"}`
- `sipplane_config_snapshot_age_seconds`

## Alternatives considered

| Option | Why not default |
|--------|-----------------|
| Per-resource revisions only | Harder atomicity; mixed Route+Trunk versions |
| Restart on config change | Violates gateway-grade UX |
| Fail SIP when CP down | Too aggressive; CP is slow path |

## Open parameters (tunable, not blockers)

- `config_stale_after` default 60s
- Whether multi-region CPs use revision epochs — **single CP region for v0.x**
