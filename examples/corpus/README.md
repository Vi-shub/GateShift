# Ingress corpus

Fixtures used by `gateshift scoreboard` and adapter regression tests.

| Tree | Source |
|------|--------|
| `public/` | Curated shapes (rewrite, rate limit, canary, auth, snippets, …) |
| `github/` | Real examples from [kubernetes/ingress-nginx](https://github.com/kubernetes/ingress-nginx) |
| `*.yaml` (root) | Extra local cases (canary pair, header-only snippets) |

```bash
gateshift scoreboard -f examples/corpus -o docs/scoreboard.md
```

Providers scored: `standard` · `envoy-gateway` · `cilium` · `istio` · `kong`.
