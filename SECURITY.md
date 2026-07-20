# Security Policy

## Supported versions

sipplane is under active development on `main` (pre-1.0). Security policy applies to:

- Code and configuration on `main`
- Published documentation and design
- Future tagged releases

| Version | Supported |
|---------|-----------|
| main | Yes (best effort) |
| pre-1.0 tags | Best effort |
| 1.x | Yes (when available) |

## Reporting a vulnerability

Please **do not** open a public GitHub Issue for security-sensitive reports.

Prefer one of:

1. GitHub **Private vulnerability advisory** on this repository (when enabled)
2. Email the maintainer via the address listed on the GitHub profile [@lixuanqun](https://github.com/lixuanqun)

Include:

- Affected component (binary, package, or doc section)
- Reproduction steps or PoC (non-destructive)
- Impact assessment (auth bypass, DoS, info leak, open relay, etc.)

We aim to acknowledge within **72 hours** and to provide a remediation plan for confirmed issues.

## Telephony-specific notes

SIP signaling systems are frequent targets for:

- Registration hijacking
- Toll fraud via open relays
- Amplification / CPS floods
- Header injection / smuggling quirks

Design and reviews assume a hostile network on the public SIP face. Default-deny routing, authenticated endpoints/trunks, and `advertised_host` requirements are intentional product goals ([RFC 0004](docs/design/rfc/0004-record-route.md)).

## Threat model

See the living draft: [docs/threat-model.md](docs/threat-model.md).

Control-plane REST supports optional Bearer token auth (`SIPPLANE_CONTROL_TOKEN`); keep the control plane off the public Internet regardless.
