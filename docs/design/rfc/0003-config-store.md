# RFC 0003 — Config store

- **Status:** Accepted default
- **Target:** P2a
- **Related:** RFC 0002; resource-model

## Problem

Architecture left “PostgreSQL vs etcd” open. Implementers need one recommended path.

## Decision

**Split responsibilities:**

| Data | Store | Why |
|------|-------|-----|
| Resources (Tenant, Route, Trunk, …) + audit | **PostgreSQL** | Relational, audit-friendly, backups, multi-tenant queries |
| Watch notification | **PostgreSQL LISTEN/NOTIFY** or **lightweight version channel** | Enough for v0.2–v0.3 fan-out |
| Location / rate counters | **Redis** (P3+) | Hot state, TTL — not config |

**etcd is optional**, not required:

- Use etcd only if an operator already runs it and wants lease/Watch semantics.
- sipplane MUST work with **Postgres-only** control plane for the default distribution.

```text
Admin API → PostgreSQL (resources, revision)
                │
                ├── NOTIFY / poll ──► Data planes (snapshot pull)
                └── Audit rows
```

## Schema sketch (non-normative)

- `resources(id, tenant, kind, name, spec jsonb, version, updated_at)`
- `revisions(revision bigint PK, hash, created_at, actor)`
- `audit(id, revision, action, actor, payload jsonb, at)`

Exact migrations land with P2a code.

## Alternatives considered

| Option | Why not default |
|--------|-----------------|
| etcd as sole store | Awkward for rich queries/audit; ops burden for SIP teams |
| Redis for config | Wrong durability/consistency story |
| Files only in prod | Drift; violates principles §2 |

## P1 implication

YAML files **must map 1:1** to resource JSON so import into Postgres is trivial at P2.
