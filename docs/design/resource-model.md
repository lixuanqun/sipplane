# Resource model

> Status: **Draft**. English canonical. 中文：[resource-model.zh-CN.md](resource-model.zh-CN.md)

sipplane configuration is a set of **versioned resources**, not a monolithic script.
Every applied set of resources has a monotonic `revision` visible to the data plane.

## Common metadata

```yaml
apiVersion: sipplane.io/v1alpha1
kind: Route
metadata:
  name: pstn-egress
  tenant: acme          # required in multi-tenant mode
  labels:
    env: prod
spec:
  # kind-specific
status:                 # server-populated later
  observedRevision: 0
```

| Field | Meaning |
|-------|---------|
| `apiVersion` | Stability channel (`v1alpha1` until first GA API) |
| `kind` | Resource type |
| `metadata.name` | Unique within tenant |
| `metadata.tenant` | Isolation key |
| `spec` | Desired state |
| `status` | Observed state (control plane) |

## Resource catalog

### Tenant

Logical isolation for credentials, routes, quotas.

```yaml
kind: Tenant
metadata:
  name: acme
spec:
  displayName: "Acme Telecom"
  quotas:
    maxEndpoints: 10000
    maxCPS: 500
```

### Endpoint

Authenticated SIP UA or PBX that may REGISTER and/or place calls.

```yaml
kind: Endpoint
metadata:
  name: alice
  tenant: acme
spec:
  aors:
    - "sip:alice@acme.example"
  auth:
    username: alice
    # secretRef points to external secret; never commit passwords
    passwordSecretRef: "secrets/acme/alice"
  allow:
    register: true
    invite: true
```

### Trunk

Interconnection toward a carrier, SBC, or peer platform.

```yaml
kind: Trunk
metadata:
  name: carrier-a
  tenant: acme
spec:
  destination:
    host: "sip.carrier-a.example"
    port: 5060
    transport: udp
  auth:
    outbound:
      username: acme
      passwordSecretRef: "secrets/acme/carrier-a"
  options:
    sendOptionsPing: true
    pingInterval: 30s
```

### Route

Match rules → actions. Evaluation order: highest `priority` first (ties by name).

```yaml
kind: Route
metadata:
  name: to-carrier
  tenant: acme
spec:
  priority: 100
  match:
    methods: ["INVITE"]
    requestUri:
      prefix: "sip:+86"
  action:
    type: loadBalance
    trunks:
      - name: carrier-a
        weight: 80
      - name: carrier-b
        weight: 20
```

Action types (planned):

| `action.type` | Behavior |
|---------------|----------|
| `proxy` | Forward to single target / URI |
| `loadBalance` | Weighted / round-robin / **Call-ID consistent hash** across `trunks` (`algorithm`) |
| `registerLookup` | Resolve Request-URI via location service |
| `reject` | Respond with configured code/reason |
| `webhook` | Ask external policy (v0.4+, timeout + fallback) |

### DispatchGroup (v0.3+)

Named backend set with health probes (OpenSIPS/Kamailio dispatcher **and** APISIX Upstream analogue).

**Schema (target YAML):** member field is `trunk` (not `ref`).

```yaml
kind: DispatchGroup
metadata:
  name: media-farm
  tenant: acme
spec:
  algorithm: consistent_hash   # round_robin | weighted | least_sessions
  hashKey: call-id
  members:
    - trunk: fs-a
      weight: 100
    - trunk: fs-b
      weight: 100
  healthCheck:
    active:
      method: OPTIONS
      interval: 30s
      timeout: 5s
    passive:
      consecutiveFailures: 5
      ejectSeconds: 30
```

**Implementation note (current):**
- LB algorithms run via Route `action.type: loadBalance` + `algorithm` (`internal/routing` → `internal/discovery`).
- Active OPTIONS probes: Trunk `spec.options.sendOptionsPing` (dataplane builds ping groups).
- YAML `kind: DispatchGroup` is the **target CRD**; control-plane loader does not ingest it yet.

Discovery sources (see [gateway-patterns.md](gateway-patterns.md)): static API, DNS SRV, later Kubernetes EndpointSlice.

### ACL / RateLimit

IP allow/deny, method filters, CPS / concurrent session caps — applied **before** routing.

Configured on the **data-plane bootstrap** (`policies:`), not as a separate CRD yet. See [docs/policies.md](../policies.md).

```yaml
# bootstrap.yaml
policies:
  acl:
    denyCidrs: ["10.255.255.0/24"]
    allowCidrs: []          # empty = allow all non-denied
    methods: []             # empty = all methods
  rateLimit:
    cps: 100
    burst: 20
```

## Lifecycle

```text
create/update via API
        │
        ▼
 validate → store → bump revision
        │
        ▼
   Watch notify data planes
        │
        ▼
 atomic snapshot swap → metrics: config_revision
```

Rollback = re-apply previous resource versions (or explicit `revision` pin in ops tooling).

## Bootstrap vs runtime

| Mode | Use |
|------|-----|
| Local YAML directory | Dev / v0.1 MVP only |
| Control-plane API | Default after v0.2 |
| GitOps (apply YAML via CI) | Same API; Git is source of intent |

Local files must map 1:1 to this resource schema so migration is trivial.

## Non-resources (intentionally)

- Media profiles / codecs — belong to media plane adapters
- Full dialplan languages — keep Route match/action small; complex logic via webhook/plugin
- Kamailio script functions — out of scope
