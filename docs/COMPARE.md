# Comparison with related tools

GateShift is a CLI migration tool (with an optional operator) focused on annotation-aware Ingress → Gateway API conversion.

[`ingress2gateway`](https://github.com/kubernetes-sigs/ingress2gateway) is a widely used structural converter for hosts, paths, backends, and TLS. The tools address overlapping but distinct needs.

## When to use which

| Job | Typical choice |
|-----|----------------|
| Quick hosts / paths / TLS → Gateway / HTTPRoute | Either tool |
| Preserve rate limits, CORS, cert-manager, affinity, canaries, snippets | GateShift |
| CI readiness score / fail-closed validate | GateShift |
| Provider-oriented Policy emission (Envoy, Cilium, Istio, Kong) | GateShift |
| GitOps PR / in-cluster `MigrationRequest` | GateShift |

## Capability matrix

| Capability | ingress2gateway | GateShift |
|------------|:---------------:|:---------:|
| Hosts / paths / backends / TLS secrets | Yes | Yes |
| L1 native filters (rewrite, redirect, CORS headers) | Partial | Yes |
| L2 provider Policies (rate limit, IP allow, timeouts, …) | Limited | Yes (richest on Envoy Gateway today) |
| Snippet handling | Generally omitted | Pattern library; residual L3 reported |
| Canary Ingress merge | Manual | Weighted HTTPRoute merge |
| Unreported annotations | Possible | Every migration annotation produces a finding |
| Readiness score | No | 0–100 + label |
| Controller capability validate | No | `validate` profiles |
| Corpus scoreboard | No | `gateshift scoreboard` |
| Targets | Gateway API YAML | `standard` · `envoy-gateway` · `cilium` · `istio` · `kong` |

## Public corpus scoreboard

```bash
make scoreboard
# → docs/scoreboard.md
```

Committed snapshot: [scoreboard.md](scoreboard.md)

Latest numbers are regenerated in [scoreboard.md](scoreboard.md) (`make scoreboard`). Corpus currently includes curated `public/`, ingress-nginx `github/`, Traefik provider fixtures, and community samples (80+ Ingress YAML files).

\*Structure-only baseline counts migration annotation keys present on fixtures (annotations that a hosts/paths/TLS-only conversion would not represent).

Envoy Gateway currently has the highest validate pass rate because L2 Policy emission and features such as session persistence / ext-auth scaffolds align with its profile. Cilium is close. Istio, Kong, and `standard` fail closed where Extended features are unsupported.

### How to read the numbers

- **Unreported (GateShift) = 0** — unknown or hard features remain L3 findings; they are not omitted.
- **Validate FAIL** on snippet/WAF fixtures is intentional (fail closed).
- **Provider columns** differ when Policy emission or the capability matrix diverges.

## Providers covered

| Target | GatewayClass (default) | Notes |
|--------|------------------------|-------|
| `envoy-gateway` | `envoy` | Primary Policy emission path today |
| `cilium` | `cilium` | Core + rewrite/regex; session persistence limited |
| `istio` | `istio` | Core + rewrite/regex; mesh Policy scaffolds |
| `kong` | `kong` | Core + rewrite; complex regex may prefer Kong plugins |
| `standard` | (portable) | Core Gateway API only |

## Recommended migration path

```text
gateshift audit -f ingress.yaml --target=envoy-gateway
gateshift convert -f ingress.yaml --target=envoy-gateway -o gateway.yaml
gateshift validate -f ingress.yaml --target=envoy-gateway
gateshift migrate -f ingress.yaml --target=envoy-gateway
```

Swap `--target` for `cilium`, `istio`, or `kong` when that controller is the destination.

## Scope notes

- GateShift does not claim 100% automatic conversion of every Ingress.
- GateShift does aim for complete annotation reporting and measurable readiness.
- Structure-only baseline estimates assume hosts/paths/TLS conversion without annotation mapping. If you measure a specific converter differently, contributions that update this document are welcome.
