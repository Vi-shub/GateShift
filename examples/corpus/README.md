# Ingress corpus

Fixtures used by `gateshift scoreboard` and adapter regression tests.

| Tree | Source | Approx. count |
|------|--------|---------------|
| `public/` | Curated GateShift shapes (incl. catalog + uncatalogued gaps) | 21 |
| `github/` | [kubernetes/ingress-nginx](https://github.com/kubernetes/ingress-nginx) docs/examples (+ local extracts) | ~20 |
| `traefik/` | [traefik/traefik](https://github.com/traefik/traefik) positive ingress-nginx fixtures | ~100 |
| `traefik-edge/` | Traefik negative / invalid / missing-secret cases | ~20 |
| `community/` | [ingress2keg](https://github.com/log1cb0mb/ingress2keg) + [ing-switch](https://github.com/saiyam1814/ing-switch) | ~12 |
| `*.yaml` (root) | Extra local cases | 2 |

How to read results: [docs/SCOREBOARD.md](../../docs/SCOREBOARD.md) · gap analysis: [docs/CORPUS_GAPS.md](../../docs/CORPUS_GAPS.md)

```bash
gateshift scoreboard -f examples/corpus -o docs/scoreboard.md
# or: make scoreboard
```

Providers scored: `standard` · `envoy-gateway` · `cilium` · `istio` · `kong`.

Non-Ingress YAML (Deployments, Services) under the tree is skipped automatically.

## Where to find more public fixtures / demos

| Source | What to harvest |
|--------|-----------------|
| [ingress-nginx `docs/examples`](https://github.com/kubernetes/ingress-nginx/tree/main/docs/examples) | Official rewrite, affinity, auth, TLS, canary, gRPC |
| [Traefik ingress-nginx fixtures](https://github.com/traefik/traefik/tree/master/pkg/provider/kubernetes/ingress-nginx/fixtures/ingresses) | ~100 single-annotation unit fixtures |
| [log1cb0mb/ingress2keg](https://github.com/log1cb0mb/ingress2keg) | Comprehensive annotation → Gateway API sample |
| [saiyam1814/ing-switch](https://github.com/saiyam1814/ing-switch) `examples/` | Scenario suite (auth, canary, CORS, gRPC, full-featured) |
| [cert-manager Ingress docs](https://cert-manager.io/docs/usage/ingress/) | Issuer annotations |
| [Bitnami charts](https://github.com/bitnami/charts) / [Artifact Hub](https://artifacthub.io/) | Chart Ingresses with nginx annotations |
| [stefanprodan/podinfo](https://github.com/stefanprodan/podinfo) | App demo (`examples/demo-podinfo`) |
| [Envoy Gateway examples](https://github.com/envoyproxy/gateway/tree/main/examples) | Target Policy shapes |
| GitHub code search | `nginx.ingress.kubernetes.io/limit-rps filename:*.yaml` |

In-repo demos (E2E, not only scoreboard):

- `examples/demo-podinfo/` + `scripts/demo-podinfo.sh`
- `examples/ingress-checkout.yaml` + `scripts/test-smoke.sh`
