# RFC 0004 — Record-Route, Contact, and advertised addresses

- **Status:** Accepted default
- **Target:** P1 single-node; P3/K8s notes
- **Related:** RFC 0001; architecture deployment shapes

## Problem

If sipplane inserts its **Pod IP** into Record-Route / Contact, subsequent in-dialog requests fail after reschedule, NAT, or multi-homing.

## Decision

sipplane always advertises an **`advertised_host`** (and optional port/transport), never an ephemeral container IP as the sole routable address.

| Deployment | `advertised_host` |
|------------|-------------------|
| P1 lab | Explicit CLI/env: public or LAN IP / hostname |
| VM / bare metal | VIP or DNS name of the SIP edge |
| Kubernetes | Service / LB hostname or external IP; **not** `status.podIP` alone |
| Edge + core | Edge VIP for UA-facing RR; core may use internal DNS for trunk-facing |

### Record-Route

- Stateful proxy **adds** Record-Route with `sip:advertised_host[:port];lr` (transport params as needed).
- Double-RR when interface bridging requires it (document when implementing; not P1 lab default).

### Contact (Registrar)

- Store **Contact as sent by UA** (after RFC 5626 / rport handling as implemented).
- Path / Service-Route: **P4 / NAT backlog**; P1 assumes direct reachable Contact or lab hairpin.

### Config knobs (v0.1+)

```yaml
# bootstrap / data-plane
listen: "0.0.0.0:5060"
advertised_host: "sip.example.com"
advertised_port: 5060
```

Missing `advertised_host` in non-loopback bind → **refuse to start** (safe default).

## Alternatives considered

| Option | Why not |
|--------|---------|
| Auto-detect primary interface IP | Wrong behind NAT/K8s |
| Pod IP + hostNetwork always | Couples scheduling; not portable |

## Test requirement (P1)

SIPp scenario must assert Record-Route host equals `advertised_host`, not `127.0.0.1` unless that was configured.
