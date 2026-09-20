# GateShift demo (podinfo)

**Recording guide:** [docs/DEMO.md](../../docs/DEMO.md) — read that before you film.

Story: **normal app → install from release → major commands → dual-run apply**.

Upstream: [stefanprodan/podinfo](https://github.com/stefanprodan/podinfo)

## Remaining before shoot

| Status | Item |
|--------|------|
| Done | `v0.1.1` release + install assets |
| Done | EG-compatible dual-run YAML |
| **You** | Warm cluster (Envoy Gateway + `GatewayClass/envoy`) |
| **You** | `gateshift version` → **0.1.1** in WSL (Linux binary) |
| **You** | One offline rehearsal with clean `kubectl apply` |

## Manifests

| File | Purpose |
|------|---------|
| `01-app.yaml` | Namespace + Deployment + Service |
| `02-ingress.yaml` | NGINX-style Ingress (rewrite, CORS, affinity, timeouts) |

## Quick path

```bash
# Install (viewers copy this)
curl -fsSL https://raw.githubusercontent.com/Vi-shub/GateShift/main/scripts/install.sh | bash
export PATH="$HOME/bin:$PATH"

# App first (no GateShift yet)
kubectl apply -f examples/demo-podinfo/01-app.yaml
kubectl apply -f examples/demo-podinfo/02-ingress.yaml

# Commands + apply (full script in docs/DEMO.md)
mkdir -p /tmp/gs-demo
kubectl -n podinfo get ingress podinfo -o yaml > /tmp/gs-demo/ingress.yaml
gateshift audit -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway
gateshift dual-run -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway -o /tmp/gs-demo/dual-run.yaml
kubectl apply -f /tmp/gs-demo/dual-run.yaml
kubectl -n podinfo get ingress,gateway,httproute,backendtrafficpolicy
```

Payoff: **Ingress still live** + staging Gateway + `podinfo-shadow` HTTPRoute.

Rehearsal helper: `bash scripts/demo-podinfo.sh`
