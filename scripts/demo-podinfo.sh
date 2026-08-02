#!/usr/bin/env bash
# Deploy stefanprodan/podinfo + Ingress, then run GateShift.
# Ubuntu WSL:
#   cd /mnt/c/Users/smsha/Desktop/GateShift
#   export PATH=$HOME/bin:$PATH
#   bash scripts/demo-podinfo.sh

set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${HOME}/bin:/usr/local/bin:${PATH}"
NS=podinfo
GATESHIFT="$ROOT/bin/gateshift"
PF_PORT="${PF_PORT:-18081}"

log() { printf '\n==> %s\n' "$*"; }

kubectl config use-context kind-gateshift >/dev/null

if [[ ! -x "$GATESHIFT" ]] || ! "$GATESHIFT" version >/dev/null 2>&1; then
  echo "Need Linux binary: bin/gateshift (build with GOOS=linux GOARCH=amd64)" >&2
  exit 1
fi

log "Deploy podinfo app (stefanprodan/podinfo)"
kubectl apply -f "$ROOT/examples/demo-podinfo/01-app.yaml"
kubectl -n "$NS" rollout status deploy/podinfo --timeout=180s

log "Deploy Ingress"
kubectl apply -f "$ROOT/examples/demo-podinfo/02-ingress.yaml"
kubectl -n "$NS" get ingress,svc,pods

log "GateShift live audit"
"$GATESHIFT" audit --namespace "$NS" --target=envoy-gateway

log "Export Ingress + convert"
mkdir -p "$ROOT/.gateshift-e2e"
kubectl -n "$NS" get ingress podinfo -o yaml > "$ROOT/.gateshift-e2e/podinfo-ingress.yaml"
"$GATESHIFT" convert -f "$ROOT/.gateshift-e2e/podinfo-ingress.yaml" --target=envoy-gateway \
  -o "$ROOT/.gateshift-e2e/podinfo-gateway.yaml"
"$GATESHIFT" validate -f "$ROOT/.gateshift-e2e/podinfo-ingress.yaml" --target=envoy-gateway || true
"$GATESHIFT" coverage -f "$ROOT/.gateshift-e2e/podinfo-ingress.yaml"

log "Ensure Envoy GatewayClass"
kubectl apply --server-side --force-conflicts \
  -f https://github.com/envoyproxy/gateway/releases/download/v1.2.1/install.yaml >/dev/null || true
kubectl wait -n envoy-gateway-system deploy/envoy-gateway --for=condition=Available --timeout=3m || true
kubectl get gatewayclass envoy >/dev/null 2>&1 || kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
EOF

log "Apply converted Gateway + HTTPRoute only"
python3 - <<PY
import re
src = r"$ROOT/.gateshift-e2e/podinfo-gateway.yaml"
dst = r"$ROOT/.gateshift-e2e/podinfo-apply.yaml"
keep = []
for d in open(src, encoding="utf-8").read().split("---"):
    kinds = re.findall(r"(?m)^kind:\s*(\S+)\s*$", d)
    if kinds and kinds[0] in ("Gateway", "HTTPRoute"):
        keep.append(d.strip())
open(dst, "w", encoding="utf-8").write("\n---\n".join(keep) + "\n")
print("docs", len(keep))
PY
kubectl apply -f "$ROOT/.gateshift-e2e/podinfo-apply.yaml"

log "Wait for Envoy proxy service + Ready pods"
ENVOY_SVC=""
for i in $(seq 1 36); do
  ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -o name 2>/dev/null | grep -E 'podinfo-gateway|podinfo' | head -1 || true)
  if [[ -n "$ENVOY_SVC" ]]; then
    echo "found $ENVOY_SVC"
    break
  fi
  sleep 5
done
if [[ -z "$ENVOY_SVC" ]]; then
  kubectl get gateway,httproute -n "$NS" -o wide
  echo "Envoy service not ready yet; Gateway may still be programming" >&2
  exit 1
fi
# Avoid port-forward race (pod Pending) that failed the first demo run.
kubectl -n envoy-gateway-system wait --for=condition=Ready pod \
  -l gateway.envoyproxy.io/owning-gateway-name=podinfo-gateway \
  --timeout=180s 2>/dev/null \
  || kubectl -n envoy-gateway-system wait --for=condition=Ready pod --all --timeout=180s >/dev/null

kubectl -n envoy-gateway-system port-forward "$ENVOY_SVC" "${PF_PORT}:80" >/tmp/podinfo-pf.log 2>&1 &
PF_PID=$!
trap 'kill $PF_PID >/dev/null 2>&1 || true' EXIT
sleep 2

log "curl podinfo via GateShift HTTPRoute"
set +e
BODY=$(curl -sS -H 'Host: podinfo.local' "http://127.0.0.1:${PF_PORT}/")
RC=$?
set -e
echo "$BODY" | head -c 400; echo
if [[ $RC -ne 0 ]]; then
  cat /tmp/podinfo-pf.log >&2 || true
  exit 1
fi
if echo "$BODY" | grep -qi 'podinfo\|hostname\|version'; then
  log "PASS — podinfo reachable through GateShift-generated Gateway API"
else
  log "Got HTTP response; inspect body above (may still be OK)"
fi

echo ""
echo "Useful:"
echo "  kubectl -n podinfo get ingress,gateway,httproute"
echo "  $GATESHIFT audit --namespace podinfo --target=envoy-gateway"
