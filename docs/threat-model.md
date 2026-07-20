# Threat model (draft)

> Status: **Draft for pre-1.0** — living document. Not a formal audit.
> Reporting: [SECURITY.md](../SECURITY.md)

## 1. Scope

| In scope | Out of scope (v1) |
|----------|-------------------|
| SIP signaling edge (proxy, registrar, policies) | Media plane (RTP/SRTP), codecs |
| Control-plane REST + config store | Full IMS / CSCF |
| Shared location (Redis) | Browser / softphone clients themselves |
| Operator-facing Helm / bootstrap | Supply-chain of third-party images beyond documented deps |

**Trust boundary (typical):**

```text
Untrusted public network
        │ SIP / TLS
        ▼
┌───────────────────┐
│  sipplane DP      │  ← this document focuses here
│  (+ optional CP)  │
└─────────┬─────────┘
          │ private
          ▼
   Redis / Postgres / media servers / policy webhooks
```

## 2. Assets

| Asset | Confidentiality | Integrity | Availability |
|-------|-----------------|-----------|--------------|
| Endpoint credentials (Digest) | High | High | Medium |
| REGISTER bindings (location) | Medium | High | High |
| Routing / ACL config (revisioned) | Medium | High | High |
| Call metadata (access log, HEP) | High (PII) | Medium | Low |
| `advertised_host` / Path / Outbound tokens | Medium | High | Medium |
| Postgres / Redis data | High | High | High |

## 3. Actors

| Actor | Intent |
|-------|--------|
| External attacker (Internet) | Toll fraud, hijack REGISTER, DoS, open relay |
| Malicious / compromised UA | Bypass ACL, enumerate AORs, CPS flood |
| Compromised trunk peer | Inject calls, steal signaling |
| Insider / broken RBAC on CP | Tamper routes, exfiltrate secrets |
| Curious operator | Misconfig (Pod IP as Record-Route, no Digest) |

## 4. Threats & mitigations

| ID | Threat | Impact | Mitigations (current / planned) | Residual risk |
|----|--------|--------|----------------------------------|---------------|
| T1 | **Open relay** — unauthenticated INVITE to PSTN | Toll fraud | Digest on REGISTER; Route reject default; ACL/rate-limit; no “accept all” lab into prod | Misconfigured `registerLookup` + weak passwords |
| T2 | **REGISTER hijack** | Call interception | Digest (realm); TLS to protect digests in transit; Outbound/Path | Digest over UDP still sniffable without TLS |
| T3 | **Contact poisoning / NAT bypass** | Wrong media/signaling target | NAT Contact rewrite; Path; fail-closed location | Malicious public Contact still possible if auth ok |
| T4 | **Record-Route to Pod IP** | Broken mid-dialog / sticky fail | **RFC 0004** require `advertised_host` | Operator sets wrong VIP |
| T5 | **CPS / INVITE flood** | Availability | Ingress rate-limit (local or **shared Redis**); ACL; `/readyz` stale | Mis-tuned CPS; Redis outage fails closed |
| T6 | **Config injection via apply** | Route takeover | dry-run/validate; audit log; **Bearer token** on `/v1/*`; CP not on public net | Single shared token ≠ RBAC; prefer network policy + token |
| T7 | **Webhook SSRF / takeover** | Bad routing decisions | Short timeout + fallback reject; private webhook URL | Webhook must be trusted; validate TLS later |
| T8 | **Redis/Postgres exposure** | Location/config leak or wipe | Private network; auth on Redis/PG; no public bind | Documented ops responsibility |
| T9 | **HEP/OTel exfil** | Call metadata leak | Optional exporters; private collectors | Enable only on trusted networks |
| T10 | **Header smuggling / parser quirks** | Auth bypass / crash | sipgo parser; keep stack updated | Fuzzing not yet continuous |
| T11 | **Stale config after CP outage** | Wrong policy continues | last-known-good + stale → not ready | Window while still “ready” |
| T12 | **Affinity miss after pod kill** | Mid-call BYE/CANCEL fail | Call-ID hash LB + Redis location ([RFC 0001](design/rfc/0001-affinity.md)) | Without dialog store, hard kill can drop calls |

## 5. Security controls checklist (operators)

- [ ] `advertised_host` = public VIP/DNS (never Pod IP)
- [ ] Public face: TLS (or private SIP only); Digest passwords rotated
- [ ] Ingress `policies.acl` + `rateLimit` enabled in production (prefer `backend: redis` multi-pod)
- [ ] Control plane: set `SIPPLANE_CONTROL_TOKEN`; Postgres/Redis **not** Internet-reachable
- [ ] Webhook URLs only on internal service mesh / private IP
- [ ] HEP/OTel collectors private; scrub if exporting off-site
- [x] Helm: non-root + NetworkPolicy examples ([deploy-production.md](deploy-production.md), chart `networkPolicy`)
- [ ] Prefer `enable_path` / Outbound for NAT UAs

## 6. What is not claimed

- No formal penetration test or third-party audit yet.
- No multi-identity CP RBAC / mTLS (Bearer token is available; see [control-plane.md](control-plane.md)).
- No continuous SIP fuzz CI yet.
- Media security (SRTP) is the media stack’s responsibility.

## 7. Next hardening (toward GA)

1. ~~Control-plane authn (token)~~ — **Done** (Bearer); mTLS / RBAC still open  
2. ~~Shared Redis rate-limit~~ — **Done** (`backend: redis`)  
3. ~~Documented NetworkPolicy + PodSecurity for Helm~~ — **Done** ([deploy-production.md](deploy-production.md))  
4. Optional dialog store for affinity-miss recovery  
5. External security review before `v1.0.0`  
6. Pin image digests in prod overlays (operator)

## Related

- [SECURITY.md](../SECURITY.md) — vulnerability reporting  
- [docs/edge.md](edge.md) — TLS / Path / HEP / policies  
- [docs/deploy-production.md](deploy-production.md) — production Helm reference  
- [docs/design/principles.md](design/principles.md) — safe defaults  
- [RFC 0004](design/rfc/0004-record-route.md) — advertised host
