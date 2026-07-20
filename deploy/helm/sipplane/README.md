# sipplane Helm chart

## Install

```bash
helm upgrade --install sipplane ./deploy/helm/sipplane \
  --set advertisedHost=sip.example.com \
  --set service.type=LoadBalancer
```

`advertisedHost` is **required** ([RFC 0004](../../../docs/design/rfc/0004-record-route.md)). Never set it to a Pod IP.

Production overlay: [examples/deploy/production-values.yaml](../../../examples/deploy/production-values.yaml)  
Guide: [docs/deploy-production.md](../../../docs/deploy-production.md)

## Common values

| Value | Purpose |
|-------|---------|
| `advertisedHost` / `advertisedPort` | Public SIP identity |
| `config.transport` | `udp` / `tcp` / `tls` |
| `config.redisAddr` | Shared location (P3) |
| `config.controlUrl` | Watch control plane |
| `controlToken` | Bearer token for CP + DP watcher |
| `config.enablePath` | RFC 3327 Path |
| `config.hepAddr` | Homer HEP collector |
| `replicaCount` | Scale data-plane pods (use Redis + Call-ID LB) |
| `networkPolicy.enabled` | Restrict SIP/metrics ingress + egress |
| `networkPolicy.restrictEgress` | Limit egress to DNS/Redis/CP/HEP/SIP |
| `podSecurityContext` | Non-root UID 65532 (matches Dockerfile) |
| `controlPlane.enabled` | Deploy `sipplane-control` (ClusterIP) |

```bash
helm upgrade --install sipplane ./deploy/helm/sipplane \
  --set advertisedHost=sip.example.com \
  --set config.redisAddr=redis:6379 \
  --set config.controlUrl=http://sipplane-control:8090 \
  --set controlToken=change-me \
  --set controlPlane.enabled=true \
  --set networkPolicy.enabled=true \
  --set replicaCount=2
```

## Security defaults

| Feature | Default |
|---------|---------|
| `runAsNonRoot` / UID `65532` | On |
| Drop all capabilities | On |
| `readOnlyRootFilesystem` | On |
| NetworkPolicy | Off (lab); enable for prod |
| Control plane Service | ClusterIP only |

Image must include `sipplane` + `sipplane-control` binaries (root `Dockerfile`).

## TLS

Mount certificates (Secret) and set:

```yaml
config:
  transport: tls
# plus volume mounts for tls_cert_file / tls_key_file in a custom values overlay
```

Chart ConfigMap today emits Path/HEP/redis/control; extend `templates/configmap.yaml` for TLS paths as needed.

## Control plane

```yaml
controlPlane:
  enabled: true
  databaseUrl: "postgres://sipplane:sipplane@postgres:5432/sipplane?sslmode=disable"
controlToken: "long-random-secret"
config:
  controlUrl: "http://sipplane-control:8090"
```

When `enabled`, the chart creates Deployment + ClusterIP Service `*-control`.  
`/healthz` stays unauthenticated; `/v1/*` requires Bearer token when `controlToken` is set.

## Verify

```bash
kubectl get svc,networkpolicy
curl http://<lb-or-pod>:8080/readyz
```
