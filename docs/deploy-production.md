# Production reference deployment

Documented reference topology for a hardened sipplane edge (GA prep).  
Not a hosted SaaS instance — operators adapt names, TLS, and CIDRs.

## Topology

```text
                    Internet / carrier
                           │
                    LB (Call-ID hash)
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
         sipplane-0                sipplane-1     ← NetworkPolicy + non-root
              │                         │
              └──────────┬──────────────┘
                         │
         ┌───────────────┼────────────────┐
         ▼               ▼                ▼
      Redis           Control          HEP/OTel
   (location+RL)   (ClusterIP+token)   (optional)
                         │
                     Postgres
```

Front LB examples: [haproxy-callid.cfg](../examples/deploy/haproxy-callid.cfg), [envoy-callid.yaml.example](../examples/deploy/envoy-callid.yaml.example).

## Install (Helm)

```bash
# Build / push image that includes sipplane + sipplane-control (see Dockerfile)
docker build -t registry.example.com/sipplane:0.1.0 .
docker push registry.example.com/sipplane:0.1.0

TOKEN=$(openssl rand -hex 32)

helm upgrade --install sipplane ./deploy/helm/sipplane \
  -f examples/deploy/production-values.yaml \
  --set image.repository=registry.example.com/sipplane \
  --set image.tag=0.1.0 \
  --set advertisedHost=sip.example.com \
  --set controlToken="$TOKEN" \
  --set controlPlane.databaseUrl='postgres://sipplane:SECRET@postgres:5432/sipplane?sslmode=require' \
  --set config.redisAddr=redis:6379
```

Data-plane bootstrap emitted by the chart should enable shared rate-limit when Redis is present — add via ConfigMap overlay if needed:

```yaml
policies:
  acl:
    denyCidrs: []
  rateLimit:
    cps: 200
    burst: 40
    backend: redis
    key: ip
```

## Security checklist (must)

| Control | How |
|---------|-----|
| `advertisedHost` = VIP/DNS | Helm required value ([RFC 0004](design/rfc/0004-record-route.md)) |
| Non-root + drop caps | Default `podSecurityContext` / `containerSecurityContext` |
| NetworkPolicy | `networkPolicy.enabled=true` (+ `restrictEgress` in prod values) |
| CP not public | Control Service is **ClusterIP**; Bearer token required |
| Redis/Postgres private | Same namespace / private network only |
| Digest + ACL | Resource YAML + bootstrap `policies` ([policies.md](policies.md)) |

Full threat model: [threat-model.md](threat-model.md).

## Namespace Pod Security

Prefer a namespace label (Kubernetes ≥1.25):

```bash
kubectl label namespace sipplane \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/audit=restricted \
  pod-security.kubernetes.io/warn=restricted
```

Chart defaults target **restricted**-compatible settings (`runAsNonRoot`, drop ALL, no privilege escalation, RuntimeDefault seccomp).  
`readOnlyRootFilesystem: true` requires the published image (no writable scratch needed at runtime).

## Verify

```bash
kubectl -n sipplane get pods,svc,networkpolicy
kubectl -n sipplane port-forward svc/sipplane 8080:8080
curl -sf http://127.0.0.1:8080/readyz

# Control plane (token required on /v1/*)
kubectl -n sipplane port-forward svc/sipplane-control 8090:8090
curl -sf -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8090/v1/revision
```

## Related

- Helm chart: [deploy/helm/sipplane/README.md](../deploy/helm/sipplane/README.md)  
- Edge features: [edge.md](edge.md)  
- Cluster / Redis: [cluster.md](cluster.md)  
- Control plane auth: [control-plane.md](control-plane.md)
