# Production edge features (P4)

Guide for features that harden sipplane as a public or multi-tenant **signaling edge**.
Open items (RTPEngine, WSS, Wasm, Dashboard, EndpointSlice, parallel fork) remain on [ROADMAP](../ROADMAP.md).

## Checklist (landed)

| Feature | Config / usage | Status |
|---------|----------------|--------|
| SIP **TLS** | `transport: tls` + cert/key | Done (WSS open) |
| **NAT** Contact fix | automatic on REGISTER | Done |
| **Path** / **Outbound** 5626 | `enable_path`, `enable_outbound` | Done |
| **HEP** → Homer | `hep_addr` | Done |
| **Webhook** route action | `action.type: webhook` | Done |
| **302** follow | `redirect_policy` | Done |
| **OpenTelemetry** | `otel_endpoint` | Basic OTLP HTTP |
| **Helm** | `deploy/helm/sipplane` | Chart present |

---

## TLS

```yaml
# bootstrap.yaml
listen: "0.0.0.0:5061"
transport: tls
advertised_host: "sip.example.com"
advertised_port: 5061
tls_cert_file: "/etc/sipplane/tls/tls.crt"
tls_key_file: "/etc/sipplane/tls/tls.key"
```

- Requires both cert and key; otherwise the process fails fast.
- Min TLS 1.2.
- Lab test: `go test ./internal/dataplane -run TestTLS -v`
- **WSS** (WebSocket Secure) is not implemented yet.

---

## NAT / Path / Outbound

| Knob | Effect |
|------|--------|
| (always) | REGISTER Contact rewritten from Via `received`/`rport` or packet source when private/mismatched (`internal/nat`) |
| `enable_path: true` | Add Path on REGISTER 200 (RFC 3327) |
| `enable_outbound: true` | RFC 5626 flow-token / Path when UA advertises Outbound |
| `outbound_secret` | HMAC secret for flow tokens |

```yaml
enable_path: true
enable_outbound: true
outbound_secret: "change-me-in-production"
```

Env: `SIPPLANE_ENABLE_OUTBOUND=true`.

See also [interop notes](interop/README.md) for softphone REGISTER behind NAT.

---

## HEP (Homer)

```yaml
hep_addr: "127.0.0.1:9060"
hep_capture_id: 2001
# SIPPLANE_HEP_ADDR=homer.monitoring:9060
```

- UDP HEP3 exporter (`internal/hep`); empty address disables.
- Captures proxied request/response payloads on the relay path.
- Test: `go test ./internal/hep -run TestExporterSendUDP -v`

---

## Webhook routing

Route action asks an HTTP policy service (timeout 500ms + fallback):

```yaml
kind: Route
metadata:
  name: policy-edge
spec:
  match:
    methods: ["INVITE"]
  action:
    type: webhook
    target: "http://policy.svc:8080/decide"   # or extra.url
    code: 503   # fallback reject code on timeout/error
```

Webhook JSON response:

```json
{ "action": "proxy", "target": "10.0.0.8:5070" }
```

Actions: `continue` / `registerLookup` | `proxy` | `reject`.  
Docs/tests: [policies.md](policies.md) patterns · `internal/webhook` · proxy webhook tests.

---

## 302 / redirect policy

```yaml
redirect_policy: follow   # follow | passthrough | reject (default)
```

| Value | Behavior |
|-------|----------|
| `passthrough` | Return 3xx to UAC (default) |
| `follow` | Re-INVITE to Contact host:port (simple) |
| `reject` | Treat redirect as failure path in policy helpers |

Implementation: `internal/redirect` + proxy relay loop.

---

## OpenTelemetry

```yaml
otel_endpoint: "http://localhost:4318"
# SIPPLANE_OTEL_ENDPOINT=http://otel-collector:4318
```

- OTLP HTTP tracer provider (`internal/otelx`); empty = disabled (noop).
- Sampling: ParentBased 10% ratio (see code).
- SIP spans are not yet densely instrumented on every hop — enable endpoint for control-plane/process tracing first.

---

## Helm

```bash
helm upgrade --install sipplane ./deploy/helm/sipplane \
  --set advertisedHost=sip.example.com \
  --set service.type=LoadBalancer \
  --set config.redisAddr=redis:6379 \
  --set config.hepAddr=homer:9060 \
  --set config.enablePath=true
```

`advertisedHost` is **required** (RFC 0004). Details: [deploy/helm/sipplane/README.md](../deploy/helm/sipplane/README.md).  
Production reference: [deploy-production.md](deploy-production.md).

---

## Example bootstrap (edge-oriented)

```yaml
listen: "0.0.0.0:5060"
transport: udp          # or tls with certs
advertised_host: "sip.example.com"
advertised_port: 5060
http_listen: "0.0.0.0:8080"
config_dir: "/etc/sipplane"
realm: sipplane
enable_path: true
enable_outbound: true
outbound_secret: "set-me"
hep_addr: "homer:9060"
hep_capture_id: 2001
otel_endpoint: "http://otel-collector:4318"
redirect_policy: passthrough
redis_addr: "redis:6379"
control_url: "http://sipplane-control:8090"
```

File sketch: [examples/config/bootstrap-edge.yaml](../examples/config/bootstrap-edge.yaml).

---

## Still open (P4 residuals)

| Item | Notes |
|------|--------|
| WSS | After TLS |
| RTPEngine control | External media |
| K8s EndpointSlice discovery | Trunk backends |
| Wasm plugins | Prefer webhook first |
| Dashboard | API-first |
| Parallel fork | BACKLOG B1 |

## Related

- Cluster: [cluster.md](cluster.md)  
- Control plane: [control-plane.md](control-plane.md)  
- Policies: [policies.md](policies.md)  
- Testing: [testing.md](testing.md)
