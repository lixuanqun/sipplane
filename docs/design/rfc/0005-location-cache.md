# RFC 0005 — Location store & read path

- **Status:** Accepted default
- **Target:** Interface in P1; Redis backend in P3
- **Related:** RFC 0001; principles §3–§4

## Problem

Every INVITE that needs `registerLookup` may hit shared storage. Redis RTT and outages must not undefined-ly blackhole or open-relay.

## Decision

### API (P1)

```text
LocationStore interface:
  Put(aor, contacts, expires) error
  Get(aor) (contacts, error)      // miss → ErrNotFound
  Delete(aor) error
```

- P1 implementation: **memory**
- P3 implementation: **Redis** (+ optional memory cache layer)

### Read path (P3+)

```text
Get(aor):
  1. local LRU/TTL cache (default TTL min(5s, remaining expires))
  2. on miss → Redis GET with timeout (default 50–100ms)
  3. on Redis timeout / error → **fail-closed**:
       - registerLookup INVITE → respond **480** or **503** (configurable; default 503)
       - never invent contacts
  4. on ErrNotFound → **404/480** per Route policy (default 480)
```

### Write path

- REGISTER success → write Redis + invalidate/replace local cache entry.
- TTL on Redis key ≈ registration Expires (+ small skew).

### Rate limits

- Cluster rate-limit counters may use Redis; local token bucket allowed for single-node.
- On Redis failure for rate limit: **fail-closed** for public unauthenticated sources; authenticated internal may use last local estimate (document in policy).

## Alternatives considered

| Option | Why not default |
|--------|-----------------|
| Fail-open empty location | Fraud / wrong routing |
| No local cache | Unnecessary Redis load |
| Cache without TTL | Stale contacts after unregister |

## Metrics

- `sipplane_location_lookup_total{result="hit_local|hit_redis|miss|error"}`
- `sipplane_location_lookup_latency_seconds`
