#!/usr/bin/env bash
# Dual-run KinD smoke: keep Ingress live, apply staging Gateway + shadow HTTPRoute.
#
# Prereq (one-time from PowerShell if on WSL):
#   $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/gateshift ./cmd/gateshift
#
# Run (Ubuntu WSL or CI):
#   bash scripts/test-dual-run.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${HOME}/bin:/usr/local/bin:${PATH}"
NS=shop
PF_PORT="${PF_PORT:-18081}"
INGRESS_FILE="$ROOT/examples/ingress-checkout.yaml"
OUT_DIR="$ROOT/.gateshift-e2e"
DUAL_YAML="$OUT_DIR/dual-run.yaml"
APPLY_YAML="$OUT_DIR/dual-run-apply.yaml"

log() { printf '\n==> %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1" >&2; exit 1; }; }
need kubectl
need curl
need python3

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

log "Ensure Envoy Gateway (server-side apply)"
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

# Dummy TLS secret so HTTPS listener resolves.
if ! kubectl -n "$NS" get secret checkout-tls >/dev/null 2>&1; then
  log "Create self-signed TLS secret shop/checkout-tls"
  openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
    -keyout /tmp/checkout-dr.key -out /tmp/checkout-dr.crt \
    -subj "/CN=checkout.example.com" >/dev/null 2>&1
  kubectl -n "$NS" create secret tls checkout-tls \
    --cert=/tmp/checkout-dr.crt --key=/tmp/checkout-dr.key
fi

log "Apply backends + live Ingress (must remain after dual-run)"
kubectl apply -n "$NS" -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: checkout-svc
spec:
  selector:
    app: checkout
  ports:
    - port: 8080
      targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: checkout
spec:
  replicas: 1
  selector:
    matchLabels:
      app: checkout
  template:
    metadata:
      labels:
        app: checkout
    spec:
      containers:
        - name: echo
          image: hashicorp/http-echo:1.0
          args: ["-text=checkout-ok", "-listen=:8080"]
          ports:
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: checkout-ui
spec:
  selector:
    app: checkout-ui
  ports:
    - port: 80
      targetPort: 5678
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: checkout-ui
spec:
  replicas: 1
  selector:
    matchLabels:
      app: checkout-ui
  template:
    metadata:
      labels:
        app: checkout-ui
    spec:
      containers:
        - name: echo
          image: hashicorp/http-echo:1.0
          args: ["-text=ui-ok"]
          ports:
            - containerPort: 5678
EOF

kubectl apply -f "$INGRESS_FILE"
kubectl wait -n "$NS" deploy/checkout --for=condition=Available --timeout=2m
kubectl wait -n "$NS" deploy/checkout-ui --for=condition=Available --timeout=2m

INGRESS_UID_BEFORE=$(kubectl -n "$NS" get ingress checkout -o jsonpath='{.metadata.uid}')
INGRESS_RV_BEFORE=$(kubectl -n "$NS" get ingress checkout -o jsonpath='{.metadata.resourceVersion}')
log "Ingress before dual-run uid=${INGRESS_UID_BEFORE} rv=${INGRESS_RV_BEFORE}"

log "gateshift dual-run (stderr checklist + YAML)"
mkdir -p "$OUT_DIR"
"$GATESHIFT" dual-run -f "$INGRESS_FILE" --target=envoy-gateway -o "$DUAL_YAML"

# Fail if dual-run emitted an Ingress document.
if grep -E '^kind:[[:space:]]*Ingress[[:space:]]*$' "$DUAL_YAML" >/dev/null 2>&1; then
  echo "ERROR: dual-run YAML must not contain kind: Ingress" >&2
  exit 1
fi
if ! grep -q 'gateshift.io/mode: dual-run' "$DUAL_YAML"; then
  echo "ERROR: dual-run YAML missing gateshift.io/mode: dual-run" >&2
  exit 1
fi
if ! grep -q 'name: checkout-shadow' "$DUAL_YAML"; then
  echo "ERROR: dual-run YAML missing checkout-shadow HTTPRoute" >&2
  exit 1
fi
if ! grep -q 'name: checkout-staging-gateway' "$DUAL_YAML"; then
  echo "ERROR: dual-run YAML missing checkout-staging-gateway" >&2
  exit 1
fi

python3 - <<PY
import re
src = r"$DUAL_YAML"
dst = r"$APPLY_YAML"
keep = []
for d in open(src, encoding="utf-8").read().split("---"):
    kinds = re.findall(r"(?m)^kind:\s*(\S+)\s*$", d)
    if kinds and kinds[0] in ("Gateway", "HTTPRoute"):
        keep.append(d.strip())
if len(keep) < 2:
    raise SystemExit(f"expected Gateway+HTTPRoute, got {len(keep)}")
open(dst, "w", encoding="utf-8").write("\n---\n".join(keep) + "\n")
print("apply docs:", len(keep))
PY

log "Apply staging Gateway + shadow HTTPRoute only"
kubectl apply -f "$APPLY_YAML"

log "Assert Ingress untouched"
if ! kubectl -n "$NS" get ingress checkout >/dev/null 2>&1; then
  echo "ERROR: Ingress checkout was deleted" >&2
  exit 1
fi
INGRESS_UID_AFTER=$(kubectl -n "$NS" get ingress checkout -o jsonpath='{.metadata.uid}')
if [[ "$INGRESS_UID_BEFORE" != "$INGRESS_UID_AFTER" ]]; then
  echo "ERROR: Ingress uid changed (recreated). before=$INGRESS_UID_BEFORE after=$INGRESS_UID_AFTER" >&2
  exit 1
fi
log "Ingress still present with same uid"

log "Assert shadow resources"
kubectl -n "$NS" get gateway checkout-staging-gateway
kubectl -n "$NS" get httproute checkout-shadow
MODE=$(kubectl -n "$NS" get httproute checkout-shadow -o jsonpath='{.metadata.annotations.gateshift\.io/mode}')
if [[ "$MODE" != "dual-run" ]]; then
  echo "ERROR: expected gateshift.io/mode=dual-run on shadow route, got '$MODE'" >&2
  exit 1
fi
SHADOW=$(kubectl -n "$NS" get httproute checkout-shadow -o jsonpath='{.metadata.annotations.gateshift\.io/shadow}')
if [[ "$SHADOW" != "true" ]]; then
  echo "ERROR: expected gateshift.io/shadow=true, got '$SHADOW'" >&2
  exit 1
fi

log "Wait for Envoy proxy Service for staging Gateway"
ENVOY_SVC=""
for i in $(seq 1 36); do
  ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -o name 2>/dev/null | grep -E 'checkout-staging-gateway|staging' | head -1 || true)
  if [[ -n "$ENVOY_SVC" ]]; then
    echo "  found $ENVOY_SVC"
    break
  fi
  sleep 5
done
if [[ -z "$ENVOY_SVC" ]]; then
  # Fallback: any envoy service referencing shop namespace naming.
  ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -o name 2>/dev/null | grep -i checkout | head -1 || true)
fi
if [[ -z "$ENVOY_SVC" ]]; then
  kubectl get gateway,httproute -n "$NS" -o yaml | sed -n '1,160p'
  kubectl get svc -n envoy-gateway-system || true
  echo "ERROR: Envoy proxy Service not created for staging Gateway" >&2
  exit 1
fi

log "Port-forward ${ENVOY_SVC} -> localhost:${PF_PORT}"
kubectl -n envoy-gateway-system port-forward "$ENVOY_SVC" "${PF_PORT}:80" >/tmp/gs-dual-pf.log 2>&1 &
PF_PID=$!
cleanup() { kill "$PF_PID" >/dev/null 2>&1 || true; }
trap cleanup EXIT
sleep 2

log "curl via staging Gateway (shadow path)"
set +e
RESP=$(curl -sS -w "\nHTTP_CODE:%{http_code}\n" -H 'Host: checkout.example.com' "http://127.0.0.1:${PF_PORT}/api")
CURL_RC=$?
set -e
echo "$RESP"
if [[ $CURL_RC -ne 0 ]]; then
  echo "curl failed rc=$CURL_RC" >&2
  cat /tmp/gs-dual-pf.log >&2 || true
  exit 1
fi
if ! echo "$RESP" | grep -Eq "HTTP_CODE:(200|301|302|404|500)|checkout-ok|ui-ok"; then
  echo "Unexpected response (Envoy may still have answered)" >&2
fi

# Final Ingress check after traffic.
if ! kubectl -n "$NS" get ingress checkout >/dev/null 2>&1; then
  echo "ERROR: Ingress missing after traffic test" >&2
  exit 1
fi

log "PASS - dual-run applied shadow path; Ingress left live"
echo ""
echo "Useful commands:"
echo "  kubectl get ingress,gateway,httproute -n shop -o wide"
echo "  kubectl -n envoy-gateway-system port-forward $ENVOY_SVC ${PF_PORT}:80"
echo "  curl -H 'Host: checkout.example.com' http://127.0.0.1:${PF_PORT}/api"
echo "  $GATESHIFT dual-run -f examples/ingress-checkout.yaml --target=envoy-gateway"
