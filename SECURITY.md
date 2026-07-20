# Security Policy

## Supported versions

sipplane has not yet released implementation packages. Security policy applies to:

- Published documentation and design (supply-chain / phishing in Issues)
- Future tagged releases once code ships

| Version | Supported |
|---------|-----------|
| main (docs) | Yes |
| pre-1.0 tags | Best effort |
| 1.x | Yes (when available) |

## Reporting a vulnerability

Please **do not** open a public GitHub Issue for security-sensitive reports.

Prefer one of:

1. GitHub **Private vulnerability advisory** on this repository (when enabled)
2. Email the maintainer via the address listed on the GitHub profile [@lixuanqun](https://github.com/lixuanqun)

Include:

- Affected component / doc section
- Reproduction steps or PoC (non-destructive)
- Impact assessment (auth bypass, DoS, info leak, etc.)

We aim to acknowledge within **72 hours** and to provide a remediation plan for confirmed issues.

## Telephony-specific notes

SIP signaling systems are frequent targets for:

- Registration hijacking
- Toll fraud via open relays
- Amplification / CPS floods
- Header injection / smuggling quirks

Design reviews should assume a hostile network on the public SIP face. Default-deny routing and authenticated trunks are intentional product goals.
