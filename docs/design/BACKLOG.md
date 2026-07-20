# Backlog (explicit, post-MVP)

Items reviewed as valuable but **not** in P1–P3 critical path.
Promote via Discussion when a real user needs them.

| ID | Item | Notes | Earliest |
|----|------|-------|----------|
| B1 | Parallel fork / simultaneous ring | Needs transaction ownership rules (RFC 0001) | after P3 |
| B2 | Number transform / LCR tables | `number_transform` policy; keep Route actions small | P2b / P4 |
| B3 | NAT / Path / outbound (RFC 5626) | Move earlier if public UA REGISTER is launch use-case | P4 (or P3.1) |
| B4 | 302 / redirect handling policies | | P4 |
| B5 | Presence / MWI | Out of signaling-edge core | after v1 |
| B6 | Full dialog store in Redis | Optional HA upgrade on RFC 0001 | P3+ flag |
| B7 | Incremental config (xDS-like) | After full snapshot proven | after v0.3 |
| B8 | Multi-region control plane | Single region for v0.x | v1+ |
| B9 | Dashboard UI | API-first | P4 last |
| B10 | Kamailio cfg importer | Explicitly non-goal as promise; maybe community tool | never official |

When promoting an item into ROADMAP, link this ID in the PR.
