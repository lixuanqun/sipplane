# Interoperability matrix (template)

> Status: **Living template** for GA. Cells use: `Pass` / `Fail` / `Partial` / `Untested` / `N/A`.
> Fill via lab runs + [interop Issues](../../.github/ISSUE_TEMPLATE/interop.md). Narrative notes: [README.md](README.md).

**sipplane revision under test:** _____________  
**Date:** _____________  
**Lab owner:** _____________

## 1. Softphones / UAs

| Peer | Version | REGISTER Digest | INVITE UA↔UA | CANCEL/487 | TLS | NAT behind router | Notes / Issue |
|------|---------|-----------------|--------------|------------|-----|-------------------|---------------|
| Linphone | | Untested | Untested | Untested | Untested | Untested | |
| Zoiper | | Untested | Untested | Untested | Untested | Untested | |
| MicroSIP | | Untested | Untested | Untested | Untested | Untested | |
| SIPp | | Pass* | Pass* (Go e2e) | Pass* | Partial | N/A | *CI / examples/sipp |

\* Automated: `go test ./internal/dataplane -run 'TestHealth|TestCancel|TestTLS'`.

## 2. Application / media servers

| Peer | Version | Trunk `proxy` | `loadBalance` | OPTIONS ping | Digest on edge | Record-Route OK | Notes / Issue |
|------|---------|---------------|---------------|--------------|----------------|-----------------|---------------|
| FreeSWITCH | | Untested | Untested | Untested | Untested | Untested | |
| Asterisk (PJSIP) | | Untested | Untested | Untested | Untested | Untested | |
| LiveKit SIP | | Untested | Untested | Untested | Untested | Untested | |
| Kamailio (as peer) | | Untested | Untested | Untested | Untested | Untested | |

## 3. Protocol / feature matrix (sipplane)

| Feature | Status | Evidence |
|---------|--------|----------|
| UDP SIP | Pass | dataplane e2e |
| TCP SIP | Pass | `TestTCPListenAndOPTIONS` |
| TLS SIP | Pass | `TestTLSListenAndHandshake` |
| WSS | Untested | Not implemented |
| REGISTER + Digest | Pass | e2e + SIPp XML (+ CI smoke) |
| INVITE/ACK/BYE | Pass | e2e |
| CANCEL → 487 | Pass | `TestCancelProxiedInvite` |
| Record-Route = advertised_host | Pass | e2e assert |
| Path / Outbound | Partial | unit + registrar tests |
| Redis location | Pass | integration when Redis up |
| Control apply/dry-run/watch | Pass | api tests + e2e-control script |
| ACL / rate limit | Pass | policy tests |
| Webhook route | Pass | proxy webhook tests |
| HEP export | Pass | HEP UDP test |
| 302 follow | Partial | unit + proxy policy |
| Prometheus metrics | Pass | scrape after OPTIONS/REGISTER |
| Access log | Pass | `accesslog` unit test |

## 4. How to record a result

1. Note sipplane git SHA and peer version.  
2. Capture pcap (filter `port 5060 or 5061`).  
3. Open Issue with [interop template](../../.github/ISSUE_TEMPLATE/interop.md).  
4. Update one cell here in a PR (`Pass` + Issue link).

## 5. GA bar (proposed)

Before `v1.0.0`, aim for:

- [ ] ≥2 softphones: REGISTER + call **Pass**
- [ ] ≥1 of FreeSWITCH / Asterisk: trunk INVITE **Pass**
- [ ] TLS REGISTER or INVITE **Pass** on at least one peer
- [ ] CANCEL **Pass** (already in CI)
- [ ] No open **Fail** on rows marked “GA required” without waiver Discussion
