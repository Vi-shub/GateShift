# Annotation fidelity

GateShift’s conversion model prioritizes **explicit classification** of Ingress annotations and related behavior. Features are never omitted without a finding.

For the capability matrix and multi-provider corpus report, see [COMPARE.md](COMPARE.md) and [scoreboard.md](scoreboard.md).

## Classification model

| Level | Meaning | Behavior |
|------:|---------|----------|
| **L1** | Native Gateway API | Emit `HTTPRoute` filters where the mapping is portable |
| **L2** | Provider extension | Emit Policy / Certificate CRDs for the selected target |
| **L3** | Untranslatable | Report snippets, Lua, and similar for human review; `validate` may fail closed |

## Quality loop

1. **Corpus:** collect real Ingress YAMLs (public fixtures and cluster exports)
2. **Frequency rank:** prioritize annotations that appear most often
3. **Pattern library:** promote safe, tested snippet idioms to L1/L2 when possible
4. **Coverage:** `gateshift coverage` reports catalog gaps and unknown keys
5. **Readiness:** `gateshift audit` prints a 0-100 migration safety score

Automated invention of Envoy (or other proxy) config from arbitrary nginx/Lua is out of scope. Curated adapters and patterns with unit tests are the supported path.

## Delivery checklist

1. Installable binaries (GoReleaser) and `scripts/install.sh`
2. CI: unit tests, KinD smoke, corpus scoreboard
3. Public scoreboard across `standard`, `envoy-gateway`, `cilium`, `istio`, and `kong`

## Practical coverage targets

Fully automatic conversion of every Ingress is not a goal. A realistic fleet distribution:

- **~80%** of Ingresses: `READY` or `READY_WITH_POLICIES` (score ≥ 60)
- **~15%**: `NEEDS_REVIEW` (partial snippet, auth, or canary edge cases)
- **~5%**: `BLOCKED` (Lua, ModSecurity, exotic controllers)

## Feedback loop

```
cluster / public Ingress corpus
        │
        ▼
gateshift coverage -f ...     → unknown keys become adapter work items
        │
        ▼
pkg/patterns matchers         → promote safe idioms with unit tests
        │
        ▼
examples/corpus fixtures      → regression suite grows with each case
```

When coverage reports `[??]` for a key:

1. Add an adapter (L1/L2), or
2. Add a snippet pattern, or
3. Mark L3 permanently with clear operator guidance
