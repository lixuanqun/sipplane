# Interop notes (P1+)

sipplane is a **signaling plane**. Peers below are typical **media / application** backends or softphones sitting behind or beside it.

For failure reports use the [Interop issue template](../../.github/ISSUE_TEMPLATE/interop.md) and attach a pcap when possible.

**Pass/Fail tracker (GA):** [matrix.md](matrix.md)

## Lab topology (recommended)

```text
UA / SIPp / softphone
        │  SIP (UDP/TCP/TLS)
        ▼
   sipplane (proxy + registrar)
        │  registerLookup or trunk proxy
        ▼
 FreeSWITCH / Asterisk / LiveKit SIP / carrier
```

- Configure `advertised_host` to the address UAs use in Record-Route / Contact ([RFC 0004](../design/rfc/0004-record-route.md)).
- Do **not** point Record-Route at an ephemeral Pod IP.
- Media (RTP) stays between UA ↔ media server; sipplane does not bridge RTP.

## FreeSWITCH

| Topic | Guidance |
|-------|----------|
| Role | FS as media/B2BUA behind sipplane; sipplane owns public REGISTER edge |
| Trunk | `Route` action `proxy` / `loadBalance` to FS Sofia profile `host:port` |
| Auth | Prefer sipplane Digest for public UAs; FS may use IP ACL or gateway auth on the internal trunk |
| Codecs | Negotiated end-to-end; sipplane does not rewrite SDP except future NAT helpers |
| REGISTER | Either UAs register on sipplane (`registerLookup`) **or** FS keeps its own directory — pick one source of truth |

Minimal route sketch (resources YAML):

```yaml
action:
  type: proxy
  target: "10.0.0.20:5060"   # FreeSWITCH Sofia
```

## Asterisk

| Topic | Guidance |
|-------|----------|
| Role | Asterisk as PBX/app server; sipplane as edge proxy/registrar |
| chan_pjsip | Create an endpoint/trunk toward sipplane **or** accept INVITEs from sipplane IP |
| Identify | Match on sipplane `advertised_host` / source IP in `identify` |
| Dialplan | In-dialog BYE/re-INVITE must hairpin via Record-Route (sipplane) |

## Softphones (Zoiper, Linphone, MicroSIP)

1. Proxy / registrar: sipplane host + port (`5060` UDP lab).
2. Username / password: Endpoint from `examples/config/lab.yaml`.
3. Domain / realm: often `acme.example` / `sipplane` — match AOR host and Digest realm.
4. After REGISTER 200, place call to another registered AOR (e.g. bob).

## Automated verification

Prefer Go tests for CI; use SIPp for manual soak:

```bash
go test ./internal/dataplane -run 'TestHealthAndRegisterInviteFlow|TestCancelProxiedInvite' -v
# see examples/sipp/README.md
```

## Known gaps (document as you hit them)

- WSS / WebRTC signaling termination (TLS TCP done; WSS open)
- Parallel fork / simultaneous ring (BACKLOG B1)
- Full SIPp dual-UAC INVITE XML (use Go e2e today)
