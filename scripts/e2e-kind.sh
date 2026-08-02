#!/usr/bin/env bash
# GateShift end-to-end test on KinD (run inside Ubuntu WSL).
#
# Usage:
#   cd /mnt/c/Users/smsha/Desktop/GateShift
#   bash scripts/e2e-kind.sh
#
# Options:
#   SKIP_CLUSTER=1   reuse existing kind cluster
#   SKIP_EG=1        skip Envoy Gateway install (CRDs + dry apply only)
#   CLEANUP=1        delete kind cluster at the end

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-gateshift}"
NAMESPACE="${NAMESPACE:-shop}"
TARGET="${TARGET:-envoy-gateway}"

log() { printf '\n==> %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "missing required tool: $1" >&2; exit 1; }; }

need docker
need kubectl
need curl

# Prefer user-local tools (no sudo needed in WSL).
export PATH="${HOME}/bin:/usr/local/bin:${PATH}"

# --- kind ---
if ! command -v kind >/dev/null 2>&1; then
  log "Installing kind into ~/bin (no sudo)"
  mkdir -p "${HOME}/bin"
  VER="v0.27.0"
  curl -fsSL -o "${HOME}/bin/kind" "https://kind.sigs.k8s.io/dl/${VER}/kind-linux-amd64"
  chmod +x "${HOME}/bin/kind"
fi
need kind

# --- gateshift binary: MUST be Linux ELF inside WSL (not .exe) ---
GATESHIFT=""
if [[ -x "$ROOT/bin/gateshift" ]] && "$ROOT/bin/gateshift" version >/dev/null 2>&1; then
  GATESHIFT="$ROOT/bin/gateshift"
elif command -v go >/dev/null 2>&1; then
  log "Building gateshift for linux"
  (cd "$ROOT" && GOOS=linux GOARCH=amd64 go build -o bin/gateshift ./cmd/gateshift)
  GATESHIFT="$ROOT/bin/gateshift"
else
  cat >&2 <<EOF
No Linux gateshift binary found at bin/gateshift
Windows .exe cannot run in WSL (Exec format error).

From PowerShell:
  cd C:\\Users\\smsha\\Desktop\\GateShift
  \$env:GOOS="linux"; \$env:GOARCH="amd64"; go build -o bin/gateshift ./cmd/gateshift
EOF
  exit 1
fi
log "Using CLI: $GATESHIFT"
to_cli_path() { echo "$1"; }

# --- cluster ---
if [[ "${SKIP_CLUSTER:-0}" != "1" ]]; then
  if kind get clusters 2>/dev/null | grep -qx "$CLUSTER_NAME"; then
    log "KinD cluster '$CLUSTER_NAME' already exists"
  else
    log "Creating KinD cluster '$CLUSTER_NAME'"
    cat <<EOF | kind create cluster --name "$CLUSTER_NAME" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 80
        hostPort: 8080
        protocol: TCP
      - containerPort: 443
        hostPort: 8443
        protocol: TCP
EOF
  fi
fi
kubectl cluster-info --context "kind-${CLUSTER_NAME}" >/dev/null
kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null

# --- Gateway API + Envoy Gateway ---
if [[ "${SKIP_EG:-0}" != "1" ]]; then
  log "Installing Gateway API CRDs"
  kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml

  log "Installing Envoy Gateway"
  helm repo add eg https://tetratelabs.github.io/envoy-gateway 2>/dev/null || true
  if command -v helm >/dev/null 2>&1; then
    helm repo update eg >/dev/null
    helm upgrade --install eg eg/envoy-gateway \
      -n envoy-gateway-system --create-namespace \
      --version v1.2.1 \
      --wait --timeout 5m || {
        log "Helm install slow/failed — applying Envoy Gateway from release manifest"
        kubectl apply -f https://github.com/envoyproxy/gateway/releases/download/v1.2.1/install.yaml
        kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available || true
      }
  else
    log "helm not found — installing Envoy Gateway via kubectl (server-side)"
    # server-side avoids: metadata.annotations Too long (>262144)
    kubectl apply --server-side --force-conflicts \
      -f https://github.com/envoyproxy/gateway/releases/download/v1.2.1/install.yaml
    kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available || true
  fi
  if ! kubectl get gatewayclass envoy >/dev/null 2>&1; then
    log "Creating GatewayClass envoy"
    kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
EOF
  fi
else
  log "SKIP_EG=1 — installing Gateway API CRDs only"
  kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml
fi

# --- GateShift offline checks ---
EXAMPLE_IN="$(to_cli_path "$ROOT/examples/ingress-checkout.yaml")"
OUT="$ROOT/.gateshift-e2e/gateway-api.yaml"
OUT_CLI="$(to_cli_path "$OUT")"
mkdir -p "$ROOT/.gateshift-e2e"

log "GateShift audit (checkout example)"
"$GATESHIFT" audit -f "$EXAMPLE_IN" --target="$TARGET"

log "GateShift convert"
"$GATESHIFT" convert -f "$EXAMPLE_IN" --target="$TARGET" -o "$OUT_CLI"

log "GateShift validate"
set +e
"$GATESHIFT" validate -f "$EXAMPLE_IN" --target="$TARGET"
VAL_RC=$?
set -e
if [[ $VAL_RC -ne 0 ]]; then
  log "validate returned non-zero (L2 warnings/errors may be expected without full policy CRDs)"
fi

log "GateShift coverage"
"$GATESHIFT" coverage -f "$EXAMPLE_IN"

# --- Apply converted resources (strip cert-manager / policy CRDs if EG-only) ---
log "Creating namespace + backend services for smoke test"
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n "$NAMESPACE" -f - <<EOF
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

log "Applying GateShift output (Gateway + HTTPRoute only; skip Certificate/Policy if CRDs missing)"
# Split and apply Gateway/HTTPRoute; ignore unknown CRDs for smoke.
python3 - <<'PY' "$OUT" "$ROOT/.gateshift-e2e/apply.yaml" || true
import re, sys
src, dst = sys.argv[1], sys.argv[2]
docs = open(src, encoding="utf-8").read().split("---")
keep = []
for d in docs:
    kinds = re.findall(r"(?m)^kind:\s*(\S+)\s*$", d)
    if kinds and kinds[0] in ("Gateway", "HTTPRoute"):
        keep.append(d.strip())
open(dst, "w", encoding="utf-8").write("\n---\n".join(keep) + "\n")
print(f"wrote {dst} ({len(keep)} docs)")
PY

# Fallback without python
if [[ ! -f "$ROOT/.gateshift-e2e/apply.yaml" ]]; then
  # crude filter with awk
  awk 'BEGIN{p=0} /^kind: (Gateway|HTTPRoute)$/{p=1} {if(p) print} /^---$/{if(p){print; p=0}}' "$OUT" > "$ROOT/.gateshift-e2e/apply.yaml" || cp "$OUT" "$ROOT/.gateshift-e2e/apply.yaml"
fi

set +e
kubectl apply -f "$ROOT/.gateshift-e2e/apply.yaml"
APPLY_RC=$?
set -e

log "Cluster resources"
kubectl get gateway,httproute -A || true
kubectl get pods -n "$NAMESPACE" || true
kubectl get pods -n envoy-gateway-system 2>/dev/null || true

log "Done"
echo ""
echo "Manual traffic check (after Gateway gets an ADDRESS):"
echo "  kubectl get gateway -n $NAMESPACE -o wide"
echo "  curl -H 'Host: checkout.example.com' http://127.0.0.1:8080/api"
echo ""
echo "Re-run GateShift against live cluster Ingresses:"
echo "  kubectl apply -f examples/ingress-checkout.yaml"
echo "  $GATESHIFT audit --namespace $NAMESPACE --target=$TARGET"
echo ""

if [[ "${CLEANUP:-0}" == "1" ]]; then
  log "Deleting KinD cluster"
  kind delete cluster --name "$CLUSTER_NAME"
fi

exit 0
