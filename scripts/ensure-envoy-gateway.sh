#!/usr/bin/env bash
# Ensure Gateway API CRDs + Envoy Gateway + GatewayClass/envoy exist.
# BackendTrafficPolicy / SecurityPolicy CRDs come from Envoy Gateway — without
# this, `kubectl apply` of GateShift dual-run YAML fails on policies.
#
# Usage:
#   bash scripts/ensure-envoy-gateway.sh
#
# Env:
#   EG_VERSION   default v1.2.1
#   SKIP_EG=1    only ensure Gateway API CRDs (no Envoy Gateway)

set -euo pipefail

EG_VERSION="${EG_VERSION:-v1.2.1}"
GWAPI_VERSION="${GWAPI_VERSION:-v1.2.0}"

log() { printf '\n==> %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1" >&2; exit 1; }; }
need kubectl

# Prefer existing kubeconfig; help k3s/rke2 users who only have distro config.
if [[ -z "${KUBECONFIG:-}" ]]; then
  if [[ ! -f "${HOME}/.kube/config" ]]; then
    for cand in /etc/rancher/k3s/k3s.yaml /etc/rancher/rke2/rke2.yaml; do
      if [[ -r "$cand" ]]; then
        export KUBECONFIG="$cand"
        log "Using KUBECONFIG=$KUBECONFIG"
        break
      fi
    done
  fi
fi

log "Ensure Gateway API CRDs (${GWAPI_VERSION})"
kubectl apply -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GWAPI_VERSION}/standard-install.yaml"

if [[ "${SKIP_EG:-0}" == "1" ]]; then
  log "SKIP_EG=1 — not installing Envoy Gateway"
  exit 0
fi

if kubectl get crd backendtrafficpolicies.gateway.envoyproxy.io >/dev/null 2>&1 \
  && kubectl get deploy -n envoy-gateway-system envoy-gateway >/dev/null 2>&1; then
  log "Envoy Gateway already present (BackendTrafficPolicy CRD + deploy)"
else
  log "Install Envoy Gateway ${EG_VERSION} (provides BackendTrafficPolicy CRDs)"
  # server-side avoids: metadata.annotations Too long (>262144)
  kubectl apply --server-side --force-conflicts \
    -f "https://github.com/envoyproxy/gateway/releases/download/${EG_VERSION}/install.yaml"
fi

log "Wait for envoy-gateway controller"
kubectl wait -n envoy-gateway-system deploy/envoy-gateway \
  --for=condition=Available --timeout=5m

if ! kubectl get gatewayclass envoy >/dev/null 2>&1; then
  log "Create GatewayClass/envoy"
  kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
EOF
fi

log "Verify BackendTrafficPolicy CRD"
kubectl get crd backendtrafficpolicies.gateway.envoyproxy.io

echo ""
echo "PASS — Envoy Gateway ready (GatewayClass/envoy + BackendTrafficPolicy CRD)"
