# Backlog (explicit deferred / promoted)

Items reviewed as valuable. **Promote via Discussion** when a real user needs them; when implementing early, mark **Promoted** here and link the ROADMAP row.

| ID | Item | Notes | Status |
|----|------|-------|--------|
| B1 | Parallel fork / simultaneous ring | Needs transaction ownership rules (RFC 0001) | Deferred (after P3 / on demand) |
| B2 | Number transform / LCR tables | `internal/transform` | **Promoted** — implemented |
| B3 | NAT / Path / outbound (RFC 5626) | `internal/nat`, `internal/outbound`, registrar Path | **Promoted** — implemented |
| B4 | 302 / redirect handling policies | `internal/redirect` + proxy follow | **Promoted** — implemented |
| B5 | Presence / MWI | Out of signaling-edge core | Deferred (after v1) |
| B6 | Full dialog store in Redis | Optional HA upgrade on RFC 0001 | Deferred (P3+ flag) |
| B7 | Incremental config (xDS-like) | After full snapshot proven | Deferred (after v0.3) |
| B8 | Multi-region control plane | Single region for v0.x | Deferred (v1+) |
| B9 | Dashboard UI | API-first | Deferred (P4 last) |
| B10 | Kamailio cfg importer | Explicitly non-goal as promise; maybe community tool | Never official |

Also landed early under P4 (not original BACKLOG IDs): HEP exporter, webhook route action, Helm chart, OTel setup, SIP TLS.

When promoting an item into ROADMAP, link this ID in the PR.
