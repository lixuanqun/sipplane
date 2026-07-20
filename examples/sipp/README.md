# SIPp scenarios (P1)

## Prerequisites

1. Start sipplane with lab config:

```bash
go run ./cmd/sipplane -config examples/config/bootstrap.yaml -resources examples/config
```

2. Install [SIPp](https://github.com/SIPp/sipp) (`sipp -v`).

3. From this directory (`examples/sipp`).

Lab credentials (see `examples/config/lab.yaml`):

| User | Password | AOR |
|------|----------|-----|
| alice | alice-secret | `sip:alice@acme.example` |
| bob | bob-secret | `sip:bob@acme.example` |

`advertised_host` in bootstrap is `127.0.0.1` for loopback lab.

## Matrix

| Scenario | File / command | Asserts |
|----------|----------------|---------|
| OPTIONS | `options_ping.xml` | 200 OK |
| REGISTER + Digest | `register_alice.xml` | 401 → 200 |
| INVITE + ACK + BYE | Go test (recommended) | Record-Route host = advertised_host |
| CANCEL → 487 | Go test | Bob sees CANCEL; UAC gets 487 |

### Automated smoke

```bash
make test-sipp
# CI job `sipp` installs sip-tester and runs the same script
```

### OPTIONS

```bash
sipp -sf options_ping.xml 127.0.0.1:5060 -m 1 -trace_err
```

### REGISTER (Digest)

```bash
sipp -sf register_alice.xml 127.0.0.1:5060 -m 1 -trace_err
```

### INVITE / CANCEL / Record-Route / OPTIONS / TCP

```bash
go test ./internal/dataplane -run 'TestHealth|TestCancel|TestOPTIONS|TestTCP|TestMetrics' -v -count=1
```

Record-Route must contain configured `advertised_host` ([RFC 0004](../../docs/design/rfc/0004-record-route.md)).

## Tips

- Use `-trace_msg` / `-trace_err` when debugging.
- If digest fails, confirm realm `sipplane` and passwords match `lab.yaml`.
- For multi-host lab, set `advertised_host` to a reachable LAN IP (not Pod IP).
