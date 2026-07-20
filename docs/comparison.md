# Comparison

> How sipplane relates to adjacent projects. 中文：[comparison.zh-CN.md](comparison.zh-CN.md)

## At a glance

| Project | Language | Primary role | Config model | Best when you need |
|---------|----------|--------------|--------------|--------------------|
| **sipplane** | Go | Signaling plane (proxy / registrar / edge) | Declarative resources + control plane | Cloud-native Go ops, K8s, hot policy |
| [Kamailio](https://www.kamailio.org/) | C | SIP proxy / server | Local script + modules | Max CPS, IMS modules, huge installed base |
| [OpenSIPS](https://www.opensips.org/) | C | SIP proxy / application-leaning | Script + rich built-ins | Integrated B2BUA, LB, operator tooling |
| [sipgo](https://github.com/emiago/sipgo) | Go | SIP **library** | Code | Building any SIP service in Go |
| [diago](https://github.com/emiago/diago) | Go | Dialog + media framework | Code | Softphone / B2BUA with RTP |
| [LiveKit SIP](https://github.com/livekit/sip) | Go | SIP ↔ WebRTC bridge | LiveKit APIs | Telephony into LiveKit rooms |
| FreeSWITCH / Asterisk | C | Softswitch / PBX | Dialplan | Apps, media, IVR |

## vs Kamailio / OpenSIPS

sipplane **learns behavior and feature checklists** from both, but does **not** aim to:

- Clone the scripting language
- Match module-for-module parity on day one
- Compete on raw single-process CPS as the primary KPI

sipplane **aims** to win on:

- Horizontal scaling with shared state
- Declarative, reviewable, GitOps-friendly policy
- First-class API and observability for Go/cloud teams
- Clear signaling/media split

Use Kamailio/OpenSIPS when you need proven carrier modules tomorrow.
Use sipplane when you are building a greenfield Go platform and accept a younger maturity curve.

## vs sipgo

| | sipgo | sipplane |
|--|-------|----------|
| Abstraction | UA / Server / Client / transactions | Productized proxy + control plane |
| You write | Handlers in Go | Resources + optional plugins |
| Dependency | — | **Depends on sipgo** |

sipplane is a **consumer** of sipgo, not a fork. Stack bugs should prefer upstream fixes.

## vs LiveKit SIP

LiveKit SIP is an excellent **scene-specific bridge** (SIP trunk ↔ LiveKit room).
sipplane targets a **general signaling plane** (registrar, multi-trunk routing, edge policy) that may sit *in front of* LiveKit, FreeSWITCH, or carriers.

## vs diagox

[diagox](https://github.com/emiago/diagox) explores ingress/egress on diago/sipgo with a product shape.
sipplane focuses on **open Apache-2.0**, explicit control-plane design, and community-driven roadmap from day one.

## vs HTTP / API gateways (APISIX, Traefik, Tyk, …)

sipplane **borrows** their ops model: policy chains, hot reload, CP/DP split, discovery, labeled metrics.
It does **not** compete as an HTTP ingress. See [gateway-patterns.md](design/gateway-patterns.md).

## Summary one-liner

> **sipgo** builds SIP messages. **sipplane** runs your SIP edge as a cloud service — with gateway-grade control plane UX.
> **Kamailio/OpenSIPS** remain the gold standard for mature C proxies — sipplane is the Go-native alternative architecture, not a drop-in binary clone.
