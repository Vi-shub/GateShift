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

## Where to find more public fixtures / demos

Copy interesting Ingress YAML into `examples/corpus/github/` or `examples/corpus/public/`, then re-run the scoreboard. Prefer real upstream examples over inventing exotic cases.

| Source | What to harvest |
|--------|-----------------|
| [kubernetes/ingress-nginx](https://github.com/kubernetes/ingress-nginx) `docs/examples/` | Official rewrite, affinity, auth, TLS, canary, snippets |
| [ingress-nginx deploy/examples](https://github.com/kubernetes/ingress-nginx/tree/main/deploy) | Common deploy shapes |
| [cert-manager docs](https://cert-manager.io/docs/usage/ingress/) | `cert-manager.io/*` issuer annotations |
| [Bitnami charts](https://github.com/bitnami/charts) / [Artifact Hub](https://artifacthub.io/) | Search charts for `kind: Ingress` + nginx annotations |
| [stefanprodan/podinfo](https://github.com/stefanprodan/podinfo) | Small app already used in `examples/demo-podinfo` |
| [Gateway API examples](https://github.com/kubernetes-sigs/gateway-api/tree/main/examples) | Target shapes (not Ingress, useful for expected output) |
| [Envoy Gateway examples](https://github.com/envoyproxy/gateway/tree/main/examples) | Policy CRDs to compare L2 emission |
| [Cilium Gateway API docs](https://docs.cilium.io/en/stable/network/servicemesh/gateway-api/) | Cilium-oriented migration targets |
| [Istio Ingress → Gateway](https://istio.io/latest/docs/tasks/traffic-management/ingress/) | Mesh migration examples |
| Public GitHub code search | `nginx.ingress.kubernetes.io/rewrite-target filename:*.yaml` (and other keys) |

In-repo demos (not scoreboard corpus, but good E2E):

- `examples/demo-podinfo/` + `scripts/demo-podinfo.sh`
- `examples/ingress-checkout.yaml` + `scripts/test-smoke.sh`
