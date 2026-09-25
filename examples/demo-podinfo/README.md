# GateShift demo (podinfo)

**Recording guide:** [docs/DEMO.md](../../docs/DEMO.md)

Story: **Envoy Gateway ready → normal app → install GateShift → commands → dual-run apply**.

## Fix for “no matches for kind BackendTrafficPolicy”

That error means Envoy Gateway CRDs are missing. From the repo:

```bash
cd ~/GateShift
git pull
bash scripts/ensure-envoy-gateway.sh
kubectl apply -f /tmp/gs-demo/dual-run.yaml   # or regenerate dual-run first
kubectl -n podinfo get ingress,gateway,httproute,backendtrafficpolicy
```

## Quick path

```bash
export PATH="$HOME/bin:$PATH"
# k3s: export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

bash scripts/ensure-envoy-gateway.sh   # REQUIRED

kubectl apply -f examples/demo-podinfo/01-app.yaml
kubectl apply -f examples/demo-podinfo/02-ingress.yaml

mkdir -p /tmp/gs-demo
kubectl -n podinfo get ingress podinfo -o yaml > /tmp/gs-demo/ingress.yaml
gateshift dual-run -f /tmp/gs-demo/ingress.yaml --target=envoy-gateway -o /tmp/gs-demo/dual-run.yaml
kubectl apply -f /tmp/gs-demo/dual-run.yaml
```

Or: `bash scripts/demo-podinfo.sh` (installs EG first).
