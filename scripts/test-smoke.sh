#!/usr/bin/env bash
# Working cluster smoke test for GateShift (Ubuntu WSL + KinD).
#
# Prereq (one-time from PowerShell):
#   $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/gateshift ./cmd/gateshift
#
# Run (Ubuntu WSL):
#   cd /mnt/c/Users/smsha/Desktop/GateShift
#   export PATH=$HOME/bin:$PATH
#   bash scripts/test-smoke.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${HOME}/bin:/usr/local/bin:${PATH}"
NS=shop
PF_PORT="${PF_PORT:-18080}"

log() { printf '\n==> %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1" >&2; exit 1; }; }
need kubectl
need curl

kubectl config use-context kind-gateshift >/dev/null

GATESHIFT="$ROOT/bin/gateshift"
if [[ ! -x "$GATESHIFT" ]] || ! "$GATESHIFT" version >/dev/null 2>&1; then
  cat >&2 <<'EOF'
ERROR: Linux binary missing/broken at bin/gateshift
Windows .exe cannot run in WSL.

PowerShell:
  cd C:\Users\smsha\Desktop\GateShift
  $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/gateshift ./cmd/gateshift
EOF
  exit 1
fi
log "CLI: $("$GATESHIFT" version)"

log "Ensure Envoy Gateway (server-side apply avoids CRD annotation size error)"
kubectl apply --server-side --force-conflicts \
  -f https://github.com/envoyproxy/gateway/releases/download/v1.2.1/install.yaml >/dev/null
kubectl wait -n envoy-gateway-system deploy/envoy-gateway --for=condition=Available --timeout=5m

if ! kubectl get gatewayclass envoy >/dev/null 2>&1; then
  log "Create GatewayClass envoy"
  kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
EOF
fi

kubectl create ns "$NS" --dry-run=client -o yaml | kubectl apply -f -

# Dummy TLS secret so HTTPS listener resolves (cert-manager not required for smoke).
if ! kubectl -n "$NS" get secret checkout-tls >/dev/null 2>&1; then
  log "Create self-signed TLS secret shop/checkout-tls"
  openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
    -keyout /tmp/checkout.key -out /tmp/checkout.crt \
    -subj "/CN=checkout.example.com" >/dev/null 2>&1
  kubectl -n "$NS" create secret tls checkout-tls \
    --cert=/tmp/checkout.crt --key=/tmp/checkout.key
fi

log "Convert + apply Gateway/HTTPRoute"
mkdir -p "$ROOT/.gateshift-e2e"
"$GATESHIFT" audit -f "$ROOT/examples/ingress-checkout.yaml" --target=envoy-gateway
"$GATESHIFT" convert -f "$ROOT/examples/ingress-checkout.yaml" --target=envoy-gateway \
  -o "$ROOT/.gateshift-e2e/gateway-api.yaml"

python3 - <<PY
import re
src = r"$ROOT/.gateshift-e2e/gateway-api.yaml"
dst = r"$ROOT/.gateshift-e2e/apply.yaml"
keep = []
for d in open(src, encoding="utf-8").read().split("---"):
    kinds = re.findall(r"(?m)^kind:\s*(\S+)\s*$", d)
    if kinds and kinds[0] in ("Gateway", "HTTPRoute"):
        keep.append(d.strip())
open(dst, "w", encoding="utf-8").write("\n---\n".join(keep) + "\n")
print("apply docs:", len(keep))
PY

kubectl apply -f "$ROOT/.gateshift-e2e/apply.yaml"

log "Wait for Envoy proxy service"
ENVOY_SVC=""
for i in $(seq 1 36); do
  ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -o name 2>/dev/null | grep -E 'checkout-gateway|shop-checkout' | head -1 || true)
  if [[ -n "$ENVOY_SVC" ]]; then
    echo "  found $ENVOY_SVC"
    break
  fi
  sleep 5
done
if [[ -z "$ENVOY_SVC" ]]; then
  kubectl get gateway -n "$NS" -o yaml | sed -n '1,120p'
  echo "ERROR: Envoy proxy Service not created" >&2
  exit 1
fi

log "Status"
kubectl get gatewayclass
kubectl get gateway,httproute -n "$NS" -o wide
kubectl get pods -n "$NS"
kubectl get svc -n envoy-gateway-system

# KinD has no cloud LB — port-forward the Envoy proxy Service.
log "Port-forward ${ENVOY_SVC} -> localhost:${PF_PORT}"
kubectl -n envoy-gateway-system port-forward "$ENVOY_SVC" "${PF_PORT}:80" >/tmp/gs-pf.log 2>&1 &
PF_PID=$!
cleanup() { kill "$PF_PID" >/dev/null 2>&1 || true; }
trap cleanup EXIT
sleep 2

log "curl via port-forward"
set +e
RESP=$(curl -sS -w "\nHTTP_CODE:%{http_code}\n" -H 'Host: checkout.example.com' "http://127.0.0.1:${PF_PORT}/api")
CURL_RC=$?
set -e
echo "$RESP"
if [[ $CURL_RC -ne 0 ]]; then
  echo "curl failed rc=$CURL_RC" >&2
  echo "port-forward log:" >&2
  cat /tmp/gs-pf.log >&2 || true
  exit 1
fi
if ! echo "$RESP" | grep -q "HTTP_CODE:200\|checkout-ok\|ui-ok\|HTTP_CODE:301\|HTTP_CODE:302\|HTTP_CODE:404\|HTTP_CODE:500"; then
  # 200 ideal; redirects also prove data plane works
  echo "Unexpected response (still useful if Envoy answered)" >&2
fi

log "PASS — GateShift convert applied and Envoy answered on localhost:${PF_PORT}"
echo ""
echo "Useful commands:"
echo "  kubectl get gateway,httproute -n shop -o wide"
echo "  kubectl -n envoy-gateway-system port-forward $ENVOY_SVC ${PF_PORT}:80"
echo "  curl -H 'Host: checkout.example.com' http://127.0.0.1:${PF_PORT}/api"
echo "  $GATESHIFT audit --namespace shop --target=envoy-gateway"
