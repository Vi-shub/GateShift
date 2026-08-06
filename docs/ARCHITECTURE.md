# Architecture

This document describes GateShift’s conversion pipeline, difficulty model, and production cutover guidance.

## Ingress → Gateway API difficulty

Parsing annotation key/value strings is straightforward. **Semantic translation is not.**

Difficulty is not evenly distributed across annotations:

| Level | Difficulty | Examples | Strategy |
|------:|------------|----------|----------|
| **L1** | Easy | `rewrite-target`, `ssl-redirect`, basic CORS headers | Emit native `HTTPRoute` filters |
| **L2** | Moderate | `limit-rps`, cert-manager issuers, IP allow lists, timeouts | Emit **provider Policy CRDs** (`BackendTrafficPolicy`, `SecurityPolicy`, `Certificate`) |
| **L3** | Hard / impossible | `configuration-snippet`, `server-snippet`, Lua, complex `auth-url` | **Do not compile nginx → Envoy**. Detect, hint, block auto-apply |

Trying to build a full nginx/Lua → Envoy compiler is a trap. GateShift’s product value is **faithful classification + safe automation**, not magical 100% conversion.

## Plug-in Adapter Pattern

Do **not** grow a 100-branch `switch`. Each annotation family is an `AnnotationAdapter`:

```go
type AnnotationAdapter interface {
    Name() string
    Level() Level
    CanHandle(key string) bool
    Transform(key, value string, ctx *Context) error
}
```

Adapters write into a shared `Context` (filters, policies, certificates, findings). The registry claims keys so multi-key families (CORS, cert-manager) run once.

Why this structure wins:

1. **Testability** — each adapter has table tests
2. **Provider variance** — L2 adapters branch on `ctx.Provider`
3. **Extensibility** — Traefik/HAProxy adapters can register beside NGINX
4. **Honest UX** — L3 adapters emit actionable hints (regex over snippet text), never fake YAML

## Pipeline

```
Ingress YAML / live API
        │
        ▼
   pkg/loader | pkg/cluster
        │
        ▼
   ordered middle layer (pkg/convert + adapters + nginxquirks)
        host-index → adapters → quirks/canary → FinalizeIR
        │
        ▼
   IR MigrationBundle (gateshift.ir/v1)  ← single contract
        │
        ├─► emitters             ──► Gateway / HTTPRoute / Policy YAML
        ├─► pkg/conformance      ──► RequiredFeatures + findings
        ├─► pkg/audit + pkg/diff ──► human reports (id / fix / evidence)
        └─► pkg/gitops           ──► PR body + branch/commit (or dry-run)
```

See [MIDDLE_LAYER.md](MIDDLE_LAYER.md) for the IR finding/feature contract.

Two delivery modes share one engine:

- **CLI** (`gateshift`) — developer/CI workflow
- **Operator** (`gateshift-operator`) — watches `MigrationRequest` CRs

## Conformance matters more than pretty YAML

Valid Gateway API YAML can still be **unschedulable** on a cluster whose controller only implements Core features. Milestone 3’s capability matrix prevents “green CI, red production.”

## Recommended production cutover

1. `gateshift audit` — inventory L1/L2/L3 debt  
2. Rewrite L3 snippets (or accept exceptions)  
3. `gateshift convert --target=envoy-gateway` + `validate`  
4. `gateshift dual-run` — keep Ingress, apply staging Gateway + `*-shadow` HTTPRoute  
5. `gateshift migrate` / operator GitOps PR  
6. Flip DNS / Gateway listeners; delete Ingress last  

## What “done” means for GateShift

Not 100% automatic conversion. Done means:

- L1 is automatic and correct  
- L2 is automatic **per target provider**, with CRDs installed  
- L3 is **never silent** — every snippet becomes a PR checklist item  
- Conformance fails closed when the controller can’t execute the route  

That is how you migrate 100+ Ingresses without a weekend outage.
