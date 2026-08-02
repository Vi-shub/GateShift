# GitHub corpus fixtures

Real / official Ingress examples used to exercise GateShift.

| File | Upstream source |
|------|-----------------|
| `affinity_cookie_*.yaml` | [kubernetes/ingress-nginx](https://github.com/kubernetes/ingress-nginx) `docs/examples/affinity` |
| `auth_*.yaml` | ingress-nginx `docs/examples/auth` |
| `customization_configuration-snippets_ingress.yaml` | ingress-nginx snippet example |
| `docker-registry_*.yaml` | ingress-nginx docker-registry example |
| `tls-termination_ingress.yaml` | ingress-nginx TLS example |
| `official-rewrite-from-docs.yaml` | ingress-nginx rewrite docs |

Run:

```bash
gateshift coverage -f examples/corpus/github/<file>.yaml
gateshift audit -f examples/corpus/github/<file>.yaml --target=envoy-gateway
gateshift validate -f examples/corpus/github/<file>.yaml --target=envoy-gateway
```
