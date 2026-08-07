# Ingress-NGINX behavioral fidelity

GateShift classifies annotations (L1/L2/L3) and also detects **behavioral quirks**
documented in the Kubernetes blog:

[Before You Migrate: Five Surprising Ingress-NGINX Behaviors](https://kubernetes.io/blog/2026/02/27/ingress-nginx-before-you-migrate/)

A structurally correct Gateway conversion can still cause outages if these
semantics are ignored.

## Detected quirks

| ID | Behavior | Audit key | Preserve option |
|----|----------|-----------|-----------------|
| 1-3 | Regex is case-insensitive prefix; `use-regex` / `rewrite-target` force regex for the **host across Ingresses** | `gateshift.io/nginx-quirk/host-regex`, `.../path-as-regex` | `--preserve-nginx-regex` |
| 4 | `/my-path` → **301** `/my-path/` | `gateshift.io/nginx-quirk/trailing-slash` | `--emit-trailing-slash-redirects` |
| 5 | URL normalization (`.`, `..`, `//`) | `gateshift.io/nginx-quirk/url-normalization` | Controller-dependent (informational) |

## Commands

```bash
# Always surfaces quirk findings
gateshift audit -f ingress.yaml --target=envoy-gateway

# Approximate Ingress-NGINX regex semantics in emitted HTTPRoutes
gateshift convert -f ingress.yaml --target=envoy-gateway --preserve-nginx-regex -o gateway.yaml

# Emit explicit trailing-slash 301 redirects
gateshift convert -f ingress.yaml --target=envoy-gateway --emit-trailing-slash-redirects -o gateway.yaml
```

`--preserve-nginx-regex` rewrites affected paths to Gateway `RegularExpression`
matches shaped like `(?i)<path>.*` (or `(?i)` + existing regex body).

## Corpus

Fixtures: [`examples/corpus/blog-k8s-2026-02/`](../examples/corpus/blog-k8s-2026-02/).

## Design stance

- **Default convert** stays portable Gateway API (may change NGINX quirks).
- **Audit** always warns so migrations are conscious.
- **Preserve flags** opt into NGINX-like behavior when you need zero-surprise cutover.
