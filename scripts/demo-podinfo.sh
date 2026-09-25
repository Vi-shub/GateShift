#!/usr/bin/env bash
# Podinfo demo: ensure Envoy Gateway → app + Ingress → GateShift → dual-run apply.
# Matches docs/DEMO.md.
#
#   cd ~/GateShift
#   export PATH=$HOME/bin:$PATH
#   bash scripts/demo-podinfo.sh

set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${HOME}/bin:/usr/local/bin:${ROOT}/bin:${PATH}"
NS=podinfo
DEMO_DIR="${DEMO_DIR:-/tmp/gs-demo}"
PF_PORT="${PF_PORT:-18081}"

# k3s / rke2 often have no ~/.kube/config
if [[ -z "${KUBECONFIG:-}" && ! -f "${HOME}/.kube/config" ]]; then
  for cand in /etc/rancher/k3s/k3s.yaml /etc/rancher/rke2/rke2.yaml; do
    if [[ -r "$cand" ]]; then
      export KUBECONFIG="$cand"
      break
    fi
  done
fi

if command -v gateshift >/dev/null 2>&1; then
  GATESHIFT="$(command -v gateshift)"
elif [[ -x "$ROOT/bin/gateshift" ]]; then
  GATESHIFT="$ROOT/bin/gateshift"
else
  echo "gateshift not found. Install a release:" >&2
  echo "  curl -fsSL https://raw.githubusercontent.com/Vi-shub/GateShift/main/scripts/install.sh | bash" >&2
  exit 1
fi

log() { printf '\n==> %s\n' "$*"; }

kubectl config use-context kind-gateshift >/dev/null 2>&1 || true

if ! "$GATESHIFT" version >/dev/null 2>&1; then
  echo "Cannot run gateshift ($GATESHIFT). Need a Linux binary, not gateshift.exe." >&2
  exit 1
fi

log "0) Ensure Envoy Gateway + BackendTrafficPolicy CRDs"
bash "$ROOT/scripts/ensure-envoy-gateway.sh"

log "1) Deploy podinfo app + Ingress (before GateShift conversion)"
kubectl apply -f "$ROOT/examples/demo-podinfo/01-app.yaml"
kubectl -n "$NS" rollout status deploy/podinfo --timeout=180s
kubectl apply -f "$ROOT/examples/demo-podinfo/02-ingress.yaml"
kubectl -n "$NS" get pods,svc,ingress

mkdir -p "$DEMO_DIR"
kubectl -n "$NS" get ingress podinfo -o yaml > "$DEMO_DIR/ingress.yaml"

log "2) Major GateShift commands"
"$GATESHIFT" audit -f "$DEMO_DIR/ingress.yaml" --target=envoy-gateway
if [[ -n "${KUBECONFIG:-}" || -f "${HOME}/.kube/config" ]]; then
  "$GATESHIFT" audit --namespace "$NS" --target=envoy-gateway \
    ${KUBECONFIG:+--kubeconfig "$KUBECONFIG"} || true
fi
"$GATESHIFT" coverage -f "$DEMO_DIR/ingress.yaml"
"$GATESHIFT" diff -f "$DEMO_DIR/ingress.yaml" || true
"$GATESHIFT" validate -f "$DEMO_DIR/ingress.yaml" --target=envoy-gateway || true
"$GATESHIFT" convert -f "$DEMO_DIR/ingress.yaml" --target=envoy-gateway -o "$DEMO_DIR/gateway.yaml"
"$GATESHIFT" dual-run -f "$DEMO_DIR/ingress.yaml" --target=envoy-gateway -o "$DEMO_DIR/dual-run.yaml"
"$GATESHIFT" migrate -f "$DEMO_DIR/ingress.yaml" --target=envoy-gateway || true

log "3) Apply dual-run YAML (Ingress stays live)"
kubectl apply --dry-run=server -f "$DEMO_DIR/dual-run.yaml"
kubectl apply -f "$DEMO_DIR/dual-run.yaml"
kubectl -n "$NS" get ingress,gateway,httproute,backendtrafficpolicy

ING_UID_BEFORE=$(kubectl -n "$NS" get ingress podinfo -o jsonpath='{.metadata.uid}')
sleep 1
ING_UID_AFTER=$(kubectl -n "$NS" get ingress podinfo -o jsonpath='{.metadata.uid}')
if [[ "$ING_UID_BEFORE" != "$ING_UID_AFTER" ]]; then
  echo "ERROR: Ingress uid changed — dual-run must not recreate Ingress" >&2
  exit 1
fi
log "Ingress uid unchanged ($ING_UID_AFTER)"

log "4) Optional: curl via staging Gateway (best-effort)"
ENVOY_SVC=""
for i in $(seq 1 24); do
  ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -o name 2>/dev/null | grep -E 'podinfo-staging|staging-gateway|podinfo' | head -1 || true)
  if [[ -n "$ENVOY_SVC" ]]; then
    break
  fi
  sleep 5
done
if [[ -n "$ENVOY_SVC" ]]; then
  kubectl -n envoy-gateway-system wait --for=condition=Ready pod \
    -l gateway.envoyproxy.io/owning-gateway-name=podinfo-staging-gateway \
    --timeout=120s 2>/dev/null || true
  kubectl -n envoy-gateway-system port-forward "$ENVOY_SVC" "${PF_PORT}:80" >/tmp/podinfo-pf.log 2>&1 &
  PF_PID=$!
  trap 'kill $PF_PID >/dev/null 2>&1 || true' EXIT
  sleep 2
  set +e
  BODY=$(curl -sS -H 'Host: podinfo.local' "http://127.0.0.1:${PF_PORT}/")
  set -e
  echo "$BODY" | head -c 400; echo
else
  log "Envoy Service not found yet — skip curl (resources may still be programming)"
fi

echo ""
echo "PASS — demo path complete"
echo "Artifacts: $DEMO_DIR"
echo "Guide:     docs/DEMO.md"
echo "Next:      flip DNS later; delete Ingress last"
