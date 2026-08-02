# How GateShift becomes more robust than ingress2gateway

## The real gap

`ingress2gateway` is excellent at **structure**: hosts, paths, backends, TLS secrets → Gateway/HTTPRoute.

It is weak at **behavior**: the annotation surface that makes production Ingresses actually work.

GateShift’s bet: **never silently drop annotations**. Classify, promote when safe, fail closed when not.

## Do not train an ML model (yet)

“Learning patterns” for this problem should mean:

1. **Corpus** — collect real Ingress YAMLs (public + your clusters)
2. **Frequency rank** — which annotations appear most?
3. **Pattern library** — regex/AST matchers for common snippet idioms
4. **Coverage score** — `gateshift coverage` shows catalog gaps
5. **Readiness score** — audit prints 0–100 migration safety

Neural nets that invent Envoy config from Lua are a liability in prod. Curated patterns with tests are how you earn SRE trust.

## What to add next (priority order)

| Priority | Capability | Why it beats ingress2gateway |
|---------:|------------|------------------------------|
| P0 | Pattern library for snippets | Turns common L3 into L1/L2 instead of dropping |
| P0 | Canary merge | Two Ingresses → one weighted HTTPRoute |
| P0 | Provider Policy emission | `limit-rps` becomes real `BackendTrafficPolicy` |
| P1 | Annotation catalog + coverage CLI | Know % of the wild you handle |
| P1 | Readiness score | Gate CI: block merge if score < 60 |
| P1 | use-regex → RegularExpression | Extended path fidelity |
| P2 | Traefik / ALB / GCE adapters | Multi-controller reality |
| P2 | Live GatewayClass feature discovery | Query cluster instead of static matrix |
| P2 | Shadow traffic / dual-run helper | Safe cutover, not just YAML |
| P3 | Optional embedding search over corpus | Suggest nearest known migration recipe |

## “Work for almost every Ingress” — honest definition

You will never hit 100% automatic conversion. Aim for:

- **~80% of Ingresses**: READY or READY_WITH_POLICIES (score ≥ 60)
- **~15%**: NEEDS_REVIEW (partial snippet / auth / canary edge cases)
- **~5%**: BLOCKED (Lua, ModSecurity, exotic controllers)

That distribution migrates fleets. Chasing the last 5% with a compiler is how projects die.

## Feedback loop (how GateShift “learns”)

```
cluster/GitHub Ingress corpus
        │
        ▼
gateshift coverage -f ...     → unknown keys become adapter tickets
        │
        ▼
pkg/patterns matchers         → promote safe idioms with unit tests
        │
        ▼
examples/corpus fixtures      → regression suite grows with every customer case
```

When an unknown annotation shows up in coverage `[??]`, either:

1. Add an adapter (L1/L2), or
2. Add a pattern (snippet), or
3. Permanently mark L3 with a clear playbook link

That is the learning system.
