# Middle-layer IR contract

GateShift’s advantage over structural converters is a **typed, auditable middle layer** between Ingress and Gateway API YAML.

## Contract (`gateshift.ir/v1`)

`MigrationBundle` is the only hand-off to emitters, conformance, audit, and GitOps:

| Field | Role |
|-------|------|
| `schemaVersion` | Pin tests/emitters (`gateshift.ir/v1`) |
| `requiredFeatures` | Controller capabilities derived from IR nodes |
| `findings[]` | First-class issues: `id`, `severity`, `fixable`/`fix`, `evidence` |
| routes / policies / certs | Provider-neutral IR nodes |

Emitters must **not** re-parse Ingress annotations.

## Ordered pipeline

1. **Host-index + quirks** — cross-Ingress Ingress-NGINX semantics  
2. **Canary split** — primary vs canary Ingresses  
3. **Adapters** — L1/L2/L3 annotation plug-ins  
4. **Route build** — matches, filters, TLS listeners  
5. **Canary merge** — weighted / header backends  
6. **Quirk attach** — findings + optional preserve/emit flags  
7. **FinalizeIR** — normalize findings, sort, annotate features  

## Findings (never silent)

| ID examples | Meaning |
|-------------|---------|
| `annotation.unknown` | Migration annotation with no adapter — recorded, not dropped |
| `quirk.*` | Behavioral Ingress-NGINX semantics (`--preserve-nginx-regex`, etc.) |
| `canary.merge` | Canary Ingress folded into primary route |
| `path.regex` / `path.implementation-specific` | PathType fidelity |

`severity=block` / `status=untranslatable` fail closed in `validate`. Unknown annotations are **warn** (`requires_policy`) so coverage can grow without false hard-fails.

## Robustness tests

- **IR goldens** — `pkg/convert` `-update-goldens`  
- **Property / invariants** — schema, IDs, determinism, redirect/backend rules  
- **Fuzz** — `FuzzFromIngresses` random annotation suffixes  

## Validate from IR

`conformance.ValidateBundle` prefers `bundle.RequiredFeatures` over re-scanning YAML shapes, so feature detection and emission stay one source of truth.
