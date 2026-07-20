# Design principles

Normative guidance for contributors. If a proposal conflicts with these, update this document first via RFC.

## 1. Signaling plane, not softswitch

sipplane terminates and routes **SIP signaling**. Media (RTP) is delegated. Do not grow a general-purpose media engine inside the data plane.

## 2. Control plane is the source of truth

Runtime policy comes from versioned resources. Local files are bootstrap or developer sugar, never the long-term authority in clustered deployments.

## 3. Fast path stays local

Routing decisions use an in-process snapshot. Control-plane I/O must not sit on the INVITE critical path without a hard timeout and fallback.

## 4. State is shareable

Anything required for multi-instance correctness (at minimum REGISTER bindings) must have a store interface. Memory is an implementation, not the API.

## 5. Prefer sipgo over NIH

Protocol parsing, transactions, and transports belong in sipgo (or carefully justified thin wrappers). Product logic (routing, tenancy, API) belongs in sipplane.

## 6. Safe defaults

Open relay is a bug. Unauthenticated public INVITE acceptance must be explicit and loud in config. Fail closed when policy is missing.

## 7. Observable by default

Every release that handles SIP must expose metrics and structured logs keyed by Call-ID and config revision.

## 8. Compatibility of behavior, not of scripts

Interop with Kamailio/OpenSIPS **deployments** matters. Byte-compatible config dialects do not.

## 9. API stability is earned

`v1alpha1` may break with changelog notes. `v1` requires a deprecation window. Document breaks in ROADMAP / release notes.

## 10. Docs are part of the product

Features without architecture notes and examples are incomplete. Design-phase PRs that only improve clarity are first-class.

## 11. Learn from gateway products, not only SIP proxies

When designing features, prefer patterns proven by APISIX, Traefik, Tyk, Easegress, KrakenD, Envoy, and Caddy:

- Policy as a chain, not a monolith script
- Observability as a product surface (metrics + access log + ready probes)
- Strict control / data plane split with last-known-good
- Service discovery + active/passive health for backends

See [gateway-patterns.md](gateway-patterns.md). Implementation tracker: [gateway-checklist.md](gateway-checklist.md). SIP correctness still outranks HTTP analogies.

## 12. Critical RFCs beat tribal knowledge

Affinity, config revision, store choice, Record-Route, and location caching defaults live in [rfc/](rfc/README.md).
Code and new design docs MUST follow them or explicitly supersede via Discussion + RFC PR.
