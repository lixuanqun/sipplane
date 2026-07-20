# Contributing to sipplane

Thanks for your interest. sipplane is in the **design phase**: documentation and architecture PRs are the highest value contributions right now.

## Ways to help (today)

1. **Review docs** — architecture, resource model, comparison, roadmap clarity
2. **Open Discussions** — SIP edge war stories, HA patterns, API taste
3. **File design Issues** — missing resources, ambiguous fields, unsafe defaults
4. **Interop notes** — what Kamailio/OpenSIPS/FS setups you need to replace
5. **Later: code** — wait for P1 issues labeled `status: implementation-open`

## Ground rules

- **Apache-2.0** contributions only (see [LICENSE](LICENSE))
- Be respectful — [Code of Conduct](CODE_OF_CONDUCT.md)
- Prefer **small, reviewable PRs**
- Follow [docs/design/principles.md](docs/design/principles.md)
- Do not contradict accepted [RFCs](docs/design/rfc/README.md) without a superseding RFC PR
- Design changes that affect `v1alpha1` resources need a short RFC under `docs/design/`
- Do **not** reimplement a SIP stack; we standardize on [sipgo](https://github.com/emiago/sipgo)
- Prefer Discussions before contentious design changes

## Development (when code exists)

```bash
# placeholder — modules land in P1
git clone https://github.com/lixuanqun/sipplane.git
cd sipplane
# go test ./...
```

Until then, validate docs locally with normal Markdown preview.

## PR checklist

- [ ] Clear problem statement
- [ ] Docs updated if behavior/design changes
- [ ] No secrets committed
- [ ] Linked Issue / Discussion when applicable

## Security

Do not file public Issues for vulnerabilities. See [SECURITY.md](SECURITY.md).

## Maintainers

Initial maintainer: [@lixuanqun](https://github.com/lixuanqun)

We will expand the maintainer list as the project grows.
