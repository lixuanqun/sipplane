# RFC 0001 — Call affinity & dialog state

- **Status:** Accepted default
- **Target:** P1 design freeze; enforce in P3 cluster
- **Related:** architecture §3, §5; gateway-patterns §5.3

## Problem

Multi-instance SIP proxies must deliver in-dialog requests (ACK, BYE, re-INVITE, UPDATE) and CANCEL to a place that can correlate them. HTTP sticky cookies do not apply to UDP SIP.

## Decision

**Default topology for clustered data plane:**

1. **REGISTER location** is always **shared** (Redis from P3; memory only on single-node P1).
2. **Call affinity** uses **consistent hashing on Call-ID** at the load balancer / dispatcher in front of (or inside) sipplane.
3. **Full dialog state in Redis** is **optional** (feature flag), not required for v0.3 MVP HA.

```text
UA / Trunk
    │
    ▼
 L4 / SIP LB  ── hash(Call-ID) ──► sipplane-A / sipplane-B / …
                                      │
                                      ▼
                                   Redis location
```

## Consequences

| Scenario | Behavior |
|----------|----------|
| Same Call-ID, same hash node alive | In-dialog messages hit same process; local transaction OK |
| Hash node dies mid-call | New messages may land elsewhere; without dialog store, mid-dialog may fail until re-INVITE/recovery — **acceptable for v0.3 with documented RTO**; improve via optional dialog store or graceful drain |
| CANCEL | Must follow same affinity as INVITE (same Call-ID) |
| Parallel fork | Out of default scope (see backlog); when added, parent transaction ownership stays on hashing node |

## Non-goals (this RFC)

- Guaranteeing zero call drop on hard kill without dialog store
- Sharing transaction-layer timers across nodes (transactions stay process-local)

## Alternatives considered

| Option | Why not default |
|--------|-----------------|
| All dialogs in Redis always | Higher RTT; harder correctness; overkill for v0.3 |
| Only L4 TCP sticky | Fails for UDP; weak for mixed transports |
| Any node + full RR rewrite each hop | Complex; still needs shared route-set |

## Implementation notes

- P1: single node — affinity N/A; implement **LocationStore interface** anyway.
- P3: document LB config example (Envoy/HAProxy/hash); sipplane may expose `X-Sipplane-Node` or rely on external hash only.
- Metric: `sipplane_affinity_miss_total` (when dialog hints missing after node loss) — later.
