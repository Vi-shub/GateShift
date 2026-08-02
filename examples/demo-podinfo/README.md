# Demo project: podinfo

Upstream: [stefanprodan/podinfo](https://github.com/stefanprodan/podinfo)

Why this project:
- Small, well-known Kubernetes demo app
- Ships with Ingress (and also native Gateway API HTTPRoute) in its Helm chart
- Perfect “before → after” story for GateShift

## Manifests

| File | Purpose |
|------|---------|
| `01-app.yaml` | Deployment + Service (`ghcr.io/stefanprodan/podinfo`) |
| `02-ingress.yaml` | NGINX-style Ingress with common annotations |

## Flow

1. Deploy app + Ingress on KinD
2. `gateshift audit --namespace podinfo`
3. `gateshift convert` → Gateway / HTTPRoute
4. Apply converted YAML (Envoy Gateway) and curl
