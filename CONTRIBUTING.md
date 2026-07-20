# Contributing to sipplane

Thanks for your interest. sipplane is **implementation-active**: code and documentation PRs are both welcome. Critical defaults live in [docs/design/rfc/](docs/design/rfc/README.md) — do not contradict them without a superseding RFC.

## Ways to help

1. **Code** — fix bugs, add tests, harden P3/P4 items (see [ROADMAP.md](ROADMAP.md))
2. **Docs** — keep EN/ZH status aligned; improve examples and RFCs
3. **Discussions** — SIP edge war stories, HA patterns, API taste
4. **Issues** — bugs, missing resources, ambiguous fields; label newcomers with `good first issue`
5. **Interop notes** — Kamailio / OpenSIPS / FreeSWITCH / Asterisk / LiveKit captures

## Ground rules

- **Apache-2.0** contributions only (see [LICENSE](LICENSE))
- Be respectful — [Code of Conduct](CODE_OF_CONDUCT.md)
- Prefer **small, reviewable PRs**
- Follow [docs/design/principles.md](docs/design/principles.md)
- Do not contradict accepted [RFCs](docs/design/rfc/README.md) without a superseding RFC PR
- Design changes that affect `v1alpha1` resources need a short note under `docs/design/`
- Do **not** reimplement a SIP stack; we standardize on [sipgo](https://github.com/emiago/sipgo)
- Prefer Discussions before contentious design changes

## Development

```bash
git clone https://github.com/lixuanqun/sipplane.git
cd sipplane

# Unit tests (no external deps required)
go test ./... -count=1

# Full automation (Postgres/Redis/E2E) — see docs/testing.md
./scripts/test.sh unit          # Linux/macOS
# .\scripts\test.ps1 unit       # Windows

go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config
```

Requires **Go 1.23+** (`GOTOOLCHAIN=auto` is fine).

## PR checklist

- [ ] Clear problem statement
- [ ] Docs updated if behavior/design changes
- [ ] Tests added or updated when touching runtime paths
- [ ] No secrets committed
- [ ] Linked Issue / Discussion when applicable

## Security

Do not file public Issues for vulnerabilities. See [SECURITY.md](SECURITY.md).

## Maintainers

Initial maintainer: [@lixuanqun](https://github.com/lixuanqun)

We will expand the maintainer list as the project grows.
