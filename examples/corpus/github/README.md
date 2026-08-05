# GitHub corpus fixtures

Real / official Ingress examples used to exercise GateShift.

| File | Upstream source |
|------|-----------------|
| `affinity_cookie_*.yaml` | [kubernetes/ingress-nginx](https://github.com/kubernetes/ingress-nginx) `docs/examples/affinity` |
| `auth_*.yaml` | ingress-nginx `docs/examples/auth` |
| `customization_configuration-snippets_ingress.yaml` | ingress-nginx snippet example |
| `docker-registry_*.yaml` | ingress-nginx docker-registry example |
| `tls-termination_ingress.yaml` | ingress-nginx TLS example |
| `official-rewrite-from-docs.yaml` / `ingress-nginx-rewrite-official.yaml` | ingress-nginx rewrite docs |
| `ingress-nginx-canary-official.yaml` | ingress-nginx canary docs |
| `ingress-nginx-grpc.yaml` | ingress-nginx gRPC backend-protocol shape |
| `ingress-nginx-multi-tls_ingress-only.yaml` | ingress-nginx multi-TLS example (Ingress extracted) |
| `ingress-nginx-static-ip_nginx-ingress.yaml` | ingress-nginx static-ip Ingress |

Run:

```bash
gateshift coverage -f examples/corpus/github/<file>.yaml
gateshift audit -f examples/corpus/github/<file>.yaml --target=envoy-gateway
gateshift validate -f examples/corpus/github/<file>.yaml --target=envoy-gateway
```
